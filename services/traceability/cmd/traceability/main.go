package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"rutadeorigen/lib/catalog"

	"github.com/jackc/pgx/v5/pgxpool"

	"rutadeorigen/lib/app"
	"rutadeorigen/lib/config"
	"rutadeorigen/lib/health"
	"rutadeorigen/lib/logger"
	"rutadeorigen/traceability/internal/infrastructure/output/repositories/structured/postgres"
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

	dsn := config.String("DATABASE_URL", "")

	var checkers []health.Checker
	var closers []app.Closer

	var pool *pgxpool.Pool

	if dsn != "" {
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
	} else {
		log.Warn("DATABASE_URL not set, using in-memory repositories")
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
