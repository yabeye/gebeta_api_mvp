package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	appmiddleware "github.com/yabeye/gebeta_api_mvp/internal/middleware"

	"github.com/yabeye/gebeta_api_mvp/internal/config"
	"github.com/yabeye/gebeta_api_mvp/internal/users"
)

// run starts the HTTP server with the given handler and blocks until a
// shutdown signal arrives, then drains in-flight requests cleanly.
// It owns nothing but the http.Server itself.
func run(ctx context.Context, cfg *config.Config, logger *slog.Logger, handler http.Handler) error {
	srv := &http.Server{
		Addr:         cfg.Server.Addr(),
		Handler:      handler,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Info("starting server", "addr", srv.Addr, "env", cfg.Env)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	stopCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	select {
	case err := <-errCh:
		return fmt.Errorf("server failed to start: %w", err)
	case <-stopCtx.Done():
		logger.Info("shutdown signal received, draining connections")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("graceful shutdown failed: %w", err)
	}

	logger.Info("server stopped cleanly")
	return nil
}

// mount builds the chi router, attaches global middleware, and mounts
// every feature's routes onto it. Returns a plain http.Handler so run()
// stays decoupled from chi entirely.
func mount(cfg *config.Config, logger *slog.Logger, dbPool *pgxpool.Pool) (http.Handler, error) {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(requestLogger(logger))
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))

	r.Get("/health", healthHandler(dbPool))

	authMiddleware, err := appmiddleware.Auth(cfg.Supabase.JWKSURL)
	if err != nil {
		return nil, fmt.Errorf("setting up auth middleware: %w", err)
	}

	usersRepo := users.NewRepository(dbPool)
	usersService := users.NewService(usersRepo, logger)
	usersHandler := users.NewHandler(usersService, logger)

	r.Route("/api/v1", func(api chi.Router) {
		api.Use(authMiddleware)
		api.Mount("/users", users.Routes(usersHandler))
	})

	return r, nil
}

// newDBPool opens a pgx connection pool against Supabase's pooler and
// verifies connectivity with a Ping before returning — fail fast at
// startup rather than on the first incoming request.
func newDBPool(ctx context.Context, cfg *config.Config, logger *slog.Logger) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.DB.URL)
	if err != nil {
		return nil, fmt.Errorf("parsing database url: %w", err)
	}

	// Required for Supabase's transaction pooler (Supavisor, port 6543),
	// which doesn't support server-side prepared statements.
	poolCfg.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol

	poolCfg.MaxConns = cfg.DB.MaxOpenConns
	poolCfg.MinConns = cfg.DB.MaxIdleConns
	poolCfg.MaxConnLifetime = cfg.DB.ConnMaxLifetime
	poolCfg.MaxConnIdleTime = cfg.DB.ConnMaxIdleTime

	pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(pingCtx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("creating pool: %w", err)
	}

	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("pinging database: %w", err)
	}

	logger.Info("database connected", "max_conns", cfg.DB.MaxOpenConns)
	return pool, nil
}

func healthHandler(dbPool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()

		w.Header().Set("Content-Type", "application/json")

		if err := dbPool.Ping(ctx); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"status":"unhealthy","reason":"database unreachable"}`))
			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}
}

func requestLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			ww := middleware.NewWrapResponseWriter(
				w,
				r.ProtoMajor,
			)

			next.ServeHTTP(ww, r)

			logger.Info(
				"request handled",
				"method", r.Method,
				"path", r.URL.Path,
				"status", ww.Status(),
				"bytes", ww.BytesWritten(),
				"duration_ms", time.Since(start).Milliseconds(),
				"request_id", middleware.GetReqID(r.Context()),
			)
		})
	}
}
