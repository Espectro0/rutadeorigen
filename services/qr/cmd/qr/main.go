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

type runtime struct {
	deps     dependencies
	checkers []health.Checker
	closers  []app.Closer
	consumer *libmessaging.Consumer
}

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

	rt := &runtime{deps: dependencies{cacheTTL: cacheTTL}}
	if err := setupStorage(ctx, log, rt); err != nil {
		log.Error("failed to set up storage", "err", err)
		os.Exit(1)
	}

	h := health.New(log, 2*time.Second, rt.checkers...)
	mux := http.NewServeMux()
	h.Register(mux)

	lotEvents := registerRoutes(mux, log, messages, rt.deps)

	var brokerLost atomic.Bool
	if rt.consumer != nil {
		watchConsumer(ctx, log, rt.consumer, lotEvents.Handle, &brokerLost, stop)
	}

	params := app.Params{Config: cfg, Logger: log, Handler: mux, Health: h, Closers: rt.closers}
	if err := app.Run(ctx, params); err != nil || brokerLost.Load() {
		log.Error("service stopped with error", "err", err)
		os.Exit(1)
	}
}

func setupStorage(ctx context.Context, log *slog.Logger, rt *runtime) error {
	switch storage := config.String("STORAGE", "postgres"); storage {
	case "postgres":
		return setupPostgres(ctx, log, rt)
	case "memory":
		log.Warn("STORAGE=memory, using in-memory repositories without messaging (development only)")
		rt.deps.publicBaseURL = config.String("PUBLIC_BASE_URL", defaultBaseURL)
		return nil
	default:
		return fmt.Errorf("unknown STORAGE %q", storage)
	}
}

func setupPostgres(ctx context.Context, log *slog.Logger, rt *runtime) error {
	baseURL, err := config.Required("PUBLIC_BASE_URL")
	if err != nil {
		return err
	}
	rt.deps.publicBaseURL = baseURL

	if err := connectDatabase(ctx, log, rt); err != nil {
		return err
	}
	if err := connectRedis(ctx, log, rt); err != nil {
		return err
	}
	return connectConsumer(log, rt)
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

	rt.deps.pool = pool
	rt.checkers = append(rt.checkers, health.NewChecker("postgres", pool.Ping))
	rt.closers = append(rt.closers, func(context.Context) error {
		pool.Close()
		return nil
	})
	return nil
}

func connectRedis(ctx context.Context, log *slog.Logger, rt *runtime) error {
	url, err := config.Required("REDIS_URL")
	if err != nil {
		return err
	}

	client, err := redis.Connect(ctx, url)
	if err != nil {
		return fmt.Errorf("connect to redis: %w", err)
	}
	log.Info("connected to redis")

	rt.deps.redis = client
	rt.checkers = append(rt.checkers, health.NewChecker("redis", pingRedis(client)))
	rt.closers = append(rt.closers, func(context.Context) error {
		if err := client.Close(); err != nil {
			return fmt.Errorf("close redis: %w", err)
		}
		return nil
	})
	return nil
}

func connectConsumer(log *slog.Logger, rt *runtime) error {
	url, err := config.Required("RABBITMQ_URL")
	if err != nil {
		return err
	}

	consumer, err := libmessaging.NewConsumer(libmessaging.ConsumerConfig{
		URL:      url,
		Queue:    lotEventsQueue,
		Bindings: []string{lotEventsBinding},
	}, log)
	if err != nil {
		return fmt.Errorf("connect to rabbitmq: %w", err)
	}
	log.Info("connected to rabbitmq")

	rt.consumer = consumer
	rt.checkers = append(rt.checkers, health.NewChecker("rabbitmq", consumer.Ping))
	rt.closers = append(rt.closers, consumer.Close)
	return nil
}

func watchConsumer(ctx context.Context, log *slog.Logger, consumer *libmessaging.Consumer, handler libmessaging.Handler, lost *atomic.Bool, stop context.CancelFunc) {
	go func() {
		if err := consumer.Run(ctx, handler); err != nil {
			lost.Store(true)
			log.Error("lot events consumer stopped", "err", err)
			stop()
		}
	}()
	go func() {
		if err := <-consumer.Closed(); err != nil {
			lost.Store(true)
			log.Error("rabbitmq connection lost", "err", err)
			stop()
		}
	}()
}

func pingRedis(client *goredis.Client) func(context.Context) error {
	return func(ctx context.Context) error {
		if err := client.Ping(ctx).Err(); err != nil {
			return fmt.Errorf("ping redis: %w", err)
		}
		return nil
	}
}
