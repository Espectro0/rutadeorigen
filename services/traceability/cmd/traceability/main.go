package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"rutadeorigen/lib/app"
	"rutadeorigen/lib/config"
	"rutadeorigen/lib/health"
	"rutadeorigen/lib/logger"
	"rutadeorigen/traceability/internal/infrastructure/output/repositories/structured/postgres"
	"rutadeorigen/traceability/internal/platform/catalog"
)

func main() {
	cfg, err := config.LoadBase()
	if err != nil {
		logger.New(os.Stdout, config.String("SERVICE_NAME", "unknown"), slog.LevelInfo).
			Error("invalid configuration", "err", err)
		os.Exit(1)
	}
	log := logger.New(os.Stdout, cfg.ServiceName, cfg.LogLevel)

	messages, err := catalog.New()
	if err != nil {
		log.Error("failed to create catalog", "err", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, os.Interrupt)
	defer stop()

	var checkers []health.Checker
	var closers []app.Closer
	var pool *pgxpool.Pool

	switch storage := config.String("STORAGE", "postgres"); storage {
	case "postgres":
		dsn, err := config.Required("DATABASE_URL")
		if err != nil {
			log.Error("invalid configuration", "err", err)
			os.Exit(1)
		}

		pool, err = postgres.Connect(ctx, dsn)
		if err != nil {
			log.Error("failed to connect to database", "err", err)
			os.Exit(1)
		}
		log.Info("connected to database")

		checkers = append(checkers, health.NewChecker("postgres", pool.Ping))
		closers = append(closers, func(context.Context) error {
			pool.Close()
			return nil
		})
	case "memory":
		log.Warn("STORAGE=memory, using in-memory repositories (development only)")
	default:
		log.Error("invalid configuration", "err", fmt.Errorf("unknown STORAGE %q", storage))
		os.Exit(1)
	}

	h := health.New(log, 2*time.Second, checkers...)
	mux := http.NewServeMux()
	h.Register(mux)

	registerRoutes(mux, log, pool, messages)

	if err := app.Run(ctx, app.Params{Config: cfg, Logger: log, Handler: mux, Health: h, Closers: closers}); err != nil {
		log.Error("service stopped with error", "err", err)
		os.Exit(1)
	}
}
