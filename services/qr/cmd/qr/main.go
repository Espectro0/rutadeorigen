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

	goredis "github.com/redis/go-redis/v9"

	"rutadeorigen/lib/app"
	"rutadeorigen/lib/config"
	"rutadeorigen/lib/health"
	"rutadeorigen/lib/logger"
	libmessaging "rutadeorigen/lib/messaging"
	"rutadeorigen/qr/internal/infrastructure/output/repositories/semistructured/redis"
	"rutadeorigen/qr/internal/infrastructure/output/repositories/structured/postgres"
	"rutadeorigen/qr/internal/platform/catalog"
)

const (
	lotEventsQueue   = "qr.lot-events"
	lotEventsBinding = "lot.*"
	defaultBaseURL   = "http://localhost:8080"
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

	cacheTTL, err := config.Duration("QR_CACHE_TTL", redis.DefaultTTL)
	if err != nil {
		log.Error("invalid configuration", "err", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, os.Interrupt)
	defer stop()

	var checkers []health.Checker
	var closers []app.Closer
	var consumer *libmessaging.Consumer
	var brokerLost atomic.Bool
	deps := dependencies{cacheTTL: cacheTTL}

	switch storage := config.String("STORAGE", "postgres"); storage {
	case "postgres":
		if deps.publicBaseURL, err = config.Required("PUBLIC_BASE_URL"); err != nil {
			fatal(log, err)
		}
		dsn, err := config.Required("DATABASE_URL")
		if err != nil {
			fatal(log, err)
		}
		redisURL, err := config.Required("REDIS_URL")
		if err != nil {
			fatal(log, err)
		}
		amqpURL, err := config.Required("RABBITMQ_URL")
		if err != nil {
			fatal(log, err)
		}

		deps.pool, err = postgres.Connect(ctx, dsn)
		if err != nil {
			log.Error("failed to connect to database", "err", err)
			os.Exit(1)
		}
		log.Info("connected to database")
		checkers = append(checkers, health.NewChecker("postgres", deps.pool.Ping))
		closers = append(closers, func(context.Context) error {
			deps.pool.Close()
			return nil
		})

		deps.redis, err = redis.Connect(ctx, redisURL)
		if err != nil {
			log.Error("failed to connect to redis", "err", err)
			os.Exit(1)
		}
		log.Info("connected to redis")
		checkers = append(checkers, health.NewChecker("redis", pingRedis(deps.redis)))
		closers = append(closers, func(context.Context) error {
			return deps.redis.Close()
		})

		consumer, err = libmessaging.NewConsumer(libmessaging.ConsumerConfig{
			URL:      amqpURL,
			Queue:    lotEventsQueue,
			Bindings: []string{lotEventsBinding},
		}, log)
		if err != nil {
			log.Error("failed to connect to rabbitmq", "err", err)
			os.Exit(1)
		}
		log.Info("connected to rabbitmq")
		checkers = append(checkers, health.NewChecker("rabbitmq", consumer.Ping))
		closers = append(closers, consumer.Close)
	case "memory":
		log.Warn("STORAGE=memory, using in-memory repositories without messaging (development only)")
		deps.publicBaseURL = config.String("PUBLIC_BASE_URL", defaultBaseURL)
	default:
		fatal(log, fmt.Errorf("unknown STORAGE %q", storage))
	}

	h := health.New(log, 2*time.Second, checkers...)
	mux := http.NewServeMux()
	h.Register(mux)

	lotCreated := registerRoutes(mux, log, messages, deps)

	if consumer != nil {
		go func() {
			if err := consumer.Run(ctx, lotCreated.Handle); err != nil {
				brokerLost.Store(true)
				log.Error("lot events consumer stopped", "err", err)
				stop()
			}
		}()
		go func() {
			if err := <-consumer.Closed(); err != nil {
				brokerLost.Store(true)
				log.Error("rabbitmq connection lost", "err", err)
				stop()
			}
		}()
	}

	if err := app.Run(ctx, app.Params{Config: cfg, Logger: log, Handler: mux, Health: h, Closers: closers}); err != nil || brokerLost.Load() {
		log.Error("service stopped with error", "err", err)
		os.Exit(1)
	}
}

func pingRedis(client *goredis.Client) func(context.Context) error {
	return func(ctx context.Context) error {
		return client.Ping(ctx).Err()
	}
}

func fatal(log *slog.Logger, err error) {
	log.Error("invalid configuration", "err", err)
	os.Exit(1)
}
