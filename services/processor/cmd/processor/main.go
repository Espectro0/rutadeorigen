package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"rutadeorigen/lib/app"
	"rutadeorigen/lib/config"
	"rutadeorigen/lib/health"
	"rutadeorigen/lib/logger"
	libmessaging "rutadeorigen/lib/messaging"
	"rutadeorigen/processor/internal/infrastructure/output/external/rejectionmessages"
	"rutadeorigen/processor/internal/infrastructure/output/external/storage"
	"rutadeorigen/processor/internal/infrastructure/output/messaging"
	"rutadeorigen/processor/internal/infrastructure/output/repositories/structured/postgres"
	"rutadeorigen/processor/internal/platform/catalog"
)

type runtime struct {
	pool       *pgxpool.Pool
	stores     buckets
	checkers   []health.Checker
	closers    []app.Closer
	consumer   *libmessaging.Consumer
	brokerLost atomic.Bool
	stop       context.CancelFunc
}

func main() {
	cfg, err := config.LoadBase()
	if err != nil {
		logger.New(os.Stdout, config.String("SERVICE_NAME", "unknown"), slog.LevelInfo).
			Error("invalid configuration", "err", err)
		os.Exit(1)
	}
	log := logger.New(os.Stdout, cfg.ServiceName, cfg.LogLevel)

	processorCatalog, err := catalog.New()
	if err != nil {
		log.Error("failed to create catalog", "err", err)
		os.Exit(1)
	}

	limits, err := loadLimits()
	if err != nil {
		log.Error("invalid configuration", "err", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, os.Interrupt)
	defer stop()

	rt := &runtime{
		stores: buckets{quarantine: storage.NewMemoryQuarantine(), evidences: storage.NewMemoryEvidences()},
		stop:   stop,
	}
	if err := setupStorage(ctx, log, rt, config.String("STORAGE", "postgres")); err != nil {
		log.Error("failed to set up storage", "err", err)
		os.Exit(1)
	}

	handler := newEvidenceUploadedHandler(log, rt.pool, rt.stores, rejectionmessages.NewCatalog(processorCatalog), limits)

	h := health.New(log, 2*time.Second, rt.checkers...)
	mux := http.NewServeMux()
	h.Register(mux)

	if rt.consumer != nil {
		watchConsumer(ctx, log, rt, handler.Handle)
	}

	params := app.Params{Config: cfg, Logger: log, Handler: mux, Health: h, Closers: rt.closers}
	if err := app.Run(ctx, params); err != nil || rt.brokerLost.Load() {
		log.Error("service stopped with error", "err", err)
		os.Exit(1)
	}
}

func setupStorage(ctx context.Context, log *slog.Logger, rt *runtime, mode string) error {
	switch mode {
	case "postgres":
		return setupPostgres(ctx, log, rt)
	case "memory":
		log.Warn("STORAGE=memory, using in-memory repositories (development only)")
		return nil
	default:
		return fmt.Errorf("unknown STORAGE %q", mode)
	}
}

func setupPostgres(ctx context.Context, log *slog.Logger, rt *runtime) error {
	if err := connectDatabase(ctx, log, rt); err != nil {
		return err
	}

	amqpURL, err := config.Required("RABBITMQ_URL")
	if err != nil {
		return err
	}
	if err := connectBroker(ctx, log, rt, amqpURL); err != nil {
		return err
	}
	if err := connectGarage(rt); err != nil {
		return err
	}
	return connectConsumer(log, rt, amqpURL)
}

func connectDatabase(ctx context.Context, log *slog.Logger, rt *runtime) error {
	dsn, err := config.Required("DATABASE_URL")
	if err != nil {
		return err
	}

	pool, err := postgres.Connect(ctx, dsn)
	if err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}
	log.Info("connected to database")

	rt.pool = pool
	rt.checkers = append(rt.checkers, health.NewChecker("postgres", pool.Ping))
	rt.closers = append(rt.closers, func(context.Context) error {
		pool.Close()
		return nil
	})
	return nil
}

func connectBroker(ctx context.Context, log *slog.Logger, rt *runtime, amqpURL string) error {
	broker, err := libmessaging.NewPublisher(amqpURL)
	if err != nil {
		return fmt.Errorf("connect to rabbitmq: %w", err)
	}
	log.Info("connected to rabbitmq")

	rt.checkers = append(rt.checkers, health.NewChecker("rabbitmq", broker.Ping))
	rt.closers = append(rt.closers, broker.Close)
	go messaging.NewOutboxRelay(rt.pool, broker, log).Run(ctx)
	go stopWhenClosed(rt, log, "rabbitmq connection lost", broker.Closed())
	return nil
}

func connectGarage(rt *runtime) error {
	garageConfig, err := loadGarage()
	if err != nil {
		return err
	}

	garage, err := storage.NewGarage(garageConfig)
	if err != nil {
		return fmt.Errorf("create garage client: %w", err)
	}

	rt.checkers = append(rt.checkers, health.NewChecker("garage", garage.Ping))
	rt.stores = buckets{quarantine: garage.Quarantine, evidences: garage.Evidences}
	return nil
}

func connectConsumer(log *slog.Logger, rt *runtime, amqpURL string) error {
	consumer, err := libmessaging.NewConsumer(libmessaging.ConsumerConfig{
		URL:      amqpURL,
		Queue:    evidenceUploadedQueue,
		Bindings: consumedEvents,
		Prefetch: oneImageAtATime,
	}, log)
	if err != nil {
		return fmt.Errorf("start rabbitmq consumer: %w", err)
	}

	rt.consumer = consumer
	rt.checkers = append(rt.checkers, health.NewChecker("rabbitmq-consumer", consumer.Ping))
	rt.closers = append(rt.closers, consumer.Close)
	go stopWhenClosed(rt, log, "rabbitmq consumer connection lost", consumer.Closed())
	return nil
}

func stopWhenClosed[E any](rt *runtime, log *slog.Logger, message string, closed <-chan *E) {
	if lost := <-closed; lost != nil {
		rt.brokerLost.Store(true)
		log.Error(message, "err", lost)
		rt.stop()
	}
}

func watchConsumer(ctx context.Context, log *slog.Logger, rt *runtime, handler libmessaging.Handler) {
	go func() {
		if err := rt.consumer.Run(ctx, handler); err != nil {
			rt.brokerLost.Store(true)
			log.Error("evidence consumer stopped", "err", err)
			rt.stop()
		}
	}()
}
