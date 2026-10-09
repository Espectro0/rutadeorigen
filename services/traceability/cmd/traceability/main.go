package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"rutadeorigen/lib/app"
	"rutadeorigen/lib/config"
	"rutadeorigen/lib/health"
	"rutadeorigen/lib/logger"
)

func main() {
	cfg, err := config.LoadBase()
	if err != nil {
		logger.New(os.Stdout, config.String("SERVICE_NAME", "unknown"), slog.LevelInfo).
			Error("invalid configuration", "err", err)
		os.Exit(1)
	}
	log := logger.New(os.Stdout, cfg.ServiceName, cfg.LogLevel)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, os.Interrupt)
	defer stop()

	h := health.New(log, 2*time.Second)
	mux := http.NewServeMux()
	h.Register(mux)

	if err := app.Run(ctx, app.Params{Config: cfg, Logger: log, Handler: mux, Health: h}); err != nil {
		log.Error("service stopped with error", "err", err)
		os.Exit(1)
	}
}
