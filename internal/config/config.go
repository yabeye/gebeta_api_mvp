package config

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

// Config is the root application configuration, composed of
// feature-scoped sub-structs. Never log this struct directly —
// use Redacted() for safe logging.
type Config struct {
	Env      string `env:"APP_ENV" env-default:"development"`
	Server   ServerConfig
	DB       DatabaseConfig
	Supabase SupabaseConfig
	Firebase FirebaseConfig
	CORS     CORSConfig
}

// ServerConfig holds HTTP server settings such as host, port,
// and timeouts.
type ServerConfig struct {
	Host            string        `env:"SERVER_HOST" env-default:"0.0.0.0"`
	Port            string        `env:"SERVER_PORT" env-default:"8080"`
	ReadTimeout     time.Duration `env:"SERVER_READ_TIMEOUT" env-default:"5s"`
	WriteTimeout    time.Duration `env:"SERVER_WRITE_TIMEOUT" env-default:"10s"`
	IdleTimeout     time.Duration `env:"SERVER_IDLE_TIMEOUT" env-default:"120s"`
	ShutdownTimeout time.Duration `env:"SERVER_SHUTDOWN_TIMEOUT" env-default:"15s"`
}

// DatabaseConfig holds Postgres connection settings. URL should point
// at Supabase's pooler (Supavisor) connection string in transaction
// mode (port 6543), not the direct connection (port 5432).
type DatabaseConfig struct {
	URL             string        `env:"DATABASE_URL" env-required:"true"`
	MaxOpenConns    int32         `env:"DB_MAX_OPEN_CONNS" env-default:"20"`
	MaxIdleConns    int32         `env:"DB_MIN_IDLE_CONNS" env-default:"2"`
	ConnMaxLifetime time.Duration `env:"DB_CONN_MAX_LIFETIME" env-default:"30m"`
	ConnMaxIdleTime time.Duration `env:"DB_CONN_MAX_IDLE_TIME" env-default:"5m"`
}

// SupabaseConfig holds credentials and settings for Supabase Auth,
// Storage, and JWT verification.
type SupabaseConfig struct {
	ProjectURL     string `env:"SUPABASE_URL" env-required:"true"`
	PublishableKey string `env:"SUPABASE_PUBLISHABLE_KEY" env-required:"true"`
	SecretKey      string `env:"SUPABASE_SECRET_KEY" env-required:"true"`
	JWKSURL        string `env:"SUPABASE_JWKS_URL"`
	StorageBucket  string `env:"SUPABASE_STORAGE_BUCKET" env-default:"uploads"`
}

// FirebaseConfig holds settings for Firebase Cloud Messaging (push
// notifications).
type FirebaseConfig struct {
	ProjectID       string `env:"FIREBASE_PROJECT_ID" env-required:"true"`
	CredentialsFile string `env:"FIREBASE_CREDENTIALS_FILE" env-required:"true"`
}

// CORSConfig holds allowed origins for cross-origin requests.
type CORSConfig struct {
	AllowedOrigins []string `env:"CORS_ALLOWED_ORIGINS" env-separator:","`
}

// Load reads configuration from the environment (and, only outside
// production, from a local .env file) and validates it. It never
// logs secret values.
func Load() (*Config, error) {
	var cfg Config

	env := strings.ToLower(strings.TrimSpace(os.Getenv("APP_ENV")))
	if env == "" {
		env = "development"
	}

	switch env {
	case "production", "staging":
		// Production/staging: secrets come from real env vars injected
		// by the platform (Docker/Fly/Railway/etc). Never touch a .env
		// file here — a stray .env baked into an image is a leak waiting
		// to happen.
		if err := cleanenv.ReadEnv(&cfg); err != nil {
			return nil, fmt.Errorf("config: failed to read env: %w", err)
		}
	default:
		// Local development convenience only.
		if _, err := os.Stat(".env"); err == nil {
			if err := cleanenv.ReadConfig(".env", &cfg); err != nil {
				return nil, fmt.Errorf("config: failed to read .env: %w", err)
			}
		} else {
			if err := cleanenv.ReadEnv(&cfg); err != nil {
				return nil, fmt.Errorf("config: failed to read env: %w", err)
			}
		}
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("config: validation failed: %w", err)
	}

	log.Printf("config loaded: env=%s %s", cfg.Env, cfg.Redacted())
	return &cfg, nil
}

// validate applies fail-fast sanity checks beyond cleanenv's
// required-field enforcement. Prefer crashing at startup over
// serving traffic with a broken/insecure config.
func (c *Config) validate() error {
	c.Env = strings.ToLower(strings.TrimSpace(c.Env))
	switch c.Env {
	case "development", "staging", "production":
	default:
		return fmt.Errorf("APP_ENV must be one of development|staging|production, got %q", c.Env)
	}

	if port, err := strconv.Atoi(c.Server.Port); err != nil || port < 1 || port > 65535 {
		return fmt.Errorf("SERVER_PORT must be a valid port number, got %q", c.Server.Port)
	}

	if _, err := url.ParseRequestURI(c.Supabase.ProjectURL); err != nil {
		return fmt.Errorf("SUPABASE_URL is not a valid URL: %w", err)
	}

	// if _, err := os.Stat(c.Firebase.CredentialsFile); err != nil {
	// 	return fmt.Errorf("FIREBASE_CREDENTIALS_FILE not readable at %q: %w", c.Firebase.CredentialsFile, err)
	// }

	if c.IsProduction() {
		if !strings.HasPrefix(c.Supabase.ProjectURL, "https://") {
			return fmt.Errorf("SUPABASE_URL must use https in production")
		}
		for _, origin := range c.CORS.AllowedOrigins {
			if origin == "*" {
				return fmt.Errorf("CORS_ALLOWED_ORIGINS must not be \"*\" in production")
			}
		}
		if len(c.CORS.AllowedOrigins) == 0 {
			return fmt.Errorf("CORS_ALLOWED_ORIGINS must be set explicitly in production")
		}
	}

	return nil
}

// IsProduction returns true if the config is running in production mode.
func (c *Config) IsProduction() bool { return c.Env == "production" }

// IsDevelopment returns true if the config is running in development mode.
func (c *Config) IsDevelopment() bool { return c.Env == "development" }

// Addr returns the host:port string for http.Server.
func (c *ServerConfig) Addr() string {
	return fmt.Sprintf("%s:%s", c.Host, c.Port)
}

// Redacted returns a safe-to-log summary of the config with every
// secret masked. Always use this instead of logging Config directly
// or calling fmt.Sprintf("%+v", cfg) anywhere in the codebase.
func (c *Config) Redacted() string {
	return fmt.Sprintf(
		"server_addr=%s db_url=%s supabase_url=%s storage_bucket=%s firebase_project=%s cors_origins=%v",
		c.Server.Addr(),
		mask(c.DB.URL),
		c.Supabase.ProjectURL, // not secret, safe to log
		c.Supabase.StorageBucket,
		c.Firebase.ProjectID,
		c.CORS.AllowedOrigins,
	)
}

// mask keeps a connection string's shape recognizable in logs
// (useful for debugging "wrong host/port" issues) without ever
// printing the password.
func mask(dsn string) string {
	u, err := url.Parse(dsn)
	if err != nil {
		return "***invalid***"
	}
	if u.User != nil {
		u.User = url.UserPassword(u.User.Username(), "***")
	}
	return u.Redacted()
}
