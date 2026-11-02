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

	var checkers []health.Checker
	var closers []app.Closer
	var pool *pgxpool.Pool
	var brokerLost atomic.Bool
	var consumer *libmessaging.Consumer
	stores := buckets{quarantine: storage.NewMemoryQuarantine(), evidences: storage.NewMemoryEvidences()}

	storageMode := config.String("STORAGE", "postgres")
	switch storageMode {
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

		amqpURL, err := config.Required("RABBITMQ_URL")
		if err != nil {
			log.Error("invalid configuration", "err", err)
			os.Exit(1)
		}
		broker, err := libmessaging.NewPublisher(amqpURL)
		if err != nil {
			log.Error("failed to connect to rabbitmq", "err", err)
			os.Exit(1)
		}
		log.Info("connected to rabbitmq")

		checkers = append(checkers, health.NewChecker("rabbitmq", broker.Ping))
		closers = append(closers, broker.Close)
		go messaging.NewOutboxRelay(pool, broker, log).Run(ctx)
		go func() {
			if err := <-broker.Closed(); err != nil {
				brokerLost.Store(true)
				log.Error("rabbitmq connection lost", "err", err)
				stop()
			}
		}()

		garageConfig, err := loadGarage()
		if err != nil {
			log.Error("invalid configuration", "err", err)
			os.Exit(1)
		}
		garage, err := storage.NewGarage(garageConfig)
		if err != nil {
			log.Error("failed to create garage client", "err", err)
			os.Exit(1)
		}
		checkers = append(checkers, health.NewChecker("garage", garage.Ping))
		stores = buckets{quarantine: garage.Quarantine, evidences: garage.Evidences}

		consumer, err = libmessaging.NewConsumer(libmessaging.ConsumerConfig{
			URL:      amqpURL,
			Queue:    evidenceUploadedQueue,
			Bindings: consumedEvents,
			Prefetch: oneImageAtATime,
		}, log)
		if err != nil {
			log.Error("failed to start rabbitmq consumer", "err", err)
			os.Exit(1)
		}
		checkers = append(checkers, health.NewChecker("rabbitmq-consumer", consumer.Ping))
		closers = append(closers, consumer.Close)
		go func() {
			if err := <-consumer.Closed(); err != nil {
				brokerLost.Store(true)
				log.Error("rabbitmq consumer connection lost", "err", err)
				stop()
			}
		}()
	case "memory":
		log.Warn("STORAGE=memory, using in-memory repositories (development only)")
	default:
		log.Error("invalid configuration", "err", fmt.Errorf("unknown STORAGE %q", storageMode))
		os.Exit(1)
	}

	handler := newEvidenceUploadedHandler(log, pool, stores, rejectionmessages.NewCatalog(processorCatalog), limits)

	h := health.New(log, 2*time.Second, checkers...)
	mux := http.NewServeMux()
	h.Register(mux)

	if consumer != nil {
		go func() {
			if err := consumer.Run(ctx, handler.Handle); err != nil {
				brokerLost.Store(true)
				log.Error("evidence consumer stopped", "err", err)
				stop()
			}
		}()
	}

	if err := app.Run(ctx, app.Params{Config: cfg, Logger: log, Handler: mux, Health: h, Closers: closers}); err != nil || brokerLost.Load() {
		log.Error("service stopped with error", "err", err)
		os.Exit(1)
	}
}
