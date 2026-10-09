package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"rutadeorigen/lib/config"
	"rutadeorigen/lib/health"
	"rutadeorigen/lib/httpserver"
)

type Closer func(ctx context.Context) error

type Params struct {
	Config  config.Base
	Logger  *slog.Logger
	Handler http.Handler
	Health  *health.Health
	Closers []Closer
}

func Run(ctx context.Context, p Params) error {
	log := p.Logger
	srv := httpserver.New(p.Config.Addr(), p.Handler, log)

	if err := srv.Listen(); err != nil {
		return err
	}

	errCh := make(chan error, 1)
	go func() { errCh <- srv.Serve() }()

	p.Health.SetReady(true)
	log.Info("service started", "addr", srv.Addr())

	select {
	case <-ctx.Done():
		log.Info("shutdown signal received")
	case err := <-errCh:
		p.Health.SetReady(false)
		closeCtx, cancel := context.WithTimeout(context.Background(), p.Config.ShutdownTimeout)
		defer cancel()
		return errors.Join(
			fmt.Errorf("server stopped unexpectedly: %w", err),
			closeAll(closeCtx, p.Closers),
		)
	}

	p.Health.SetReady(false)
	time.Sleep(p.Config.ShutdownDelay)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), p.Config.ShutdownTimeout)
	defer cancel()

	var errs []error
	if err := srv.Shutdown(shutdownCtx); err != nil {
		errs = append(errs, fmt.Errorf("shutdown: %w", err))
	}
	if err := <-errCh; err != nil {
		errs = append(errs, fmt.Errorf("serve: %w", err))
	}

	errs = append(errs, closeAll(shutdownCtx, p.Closers))

	if err := errors.Join(errs...); err != nil {
		return err
	}
	log.Info("shutdown complete")
	return nil
}

func closeAll(ctx context.Context, closers []Closer) error {
	var errs []error

	for i := len(closers) - 1; i >= 0; i-- {
		if err := closers[i](ctx); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}
