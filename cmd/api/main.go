package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/lmittmann/tint"

	"github.com/yabeye/gebeta_api_mvp/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.New(slog.NewJSONHandler(os.Stderr, nil)).Error("failed to load config", "error", err)
		os.Exit(1)
	}

	logger := newLogger(cfg)
	slog.SetDefault(logger)

	ctx := context.Background()

	dbPool, err := newDBPool(ctx, cfg, logger)
	if err != nil {
		logger.Error("database connection failed", "error", err)
		os.Exit(1)
	}
	defer dbPool.Close()

	handler, err := mount(cfg, logger, dbPool)
	if err != nil {
		logger.Error("failed to mount routes", "error", err)
		os.Exit(1)
	}

	if err := run(ctx, cfg, logger, handler); err != nil {
		logger.Error("server exited with error", "error", err)
		os.Exit(1)
	}
}

// newLogger returns a structured slog logger. Production gets plain
// JSON (for log aggregators like Loki/CloudWatch/Datadog). Development
// gets a colorized, human-friendly console format via tint.
func newLogger(cfg *config.Config) *slog.Logger {
	level := slog.LevelInfo
	if cfg.IsDevelopment() {
		level = slog.LevelDebug
	}

	if cfg.IsProduction() {
		return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: level,
		}))
	}

	return slog.New(tint.NewTextHandler(os.Stdout, &tint.Options{Level: level, TimeFormat: time.Kitchen, AddSource: false}))
}
