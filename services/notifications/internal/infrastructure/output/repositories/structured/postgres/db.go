package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	connectTimeout = 10 * time.Second
	databaseZone   = "UTC"
)

func Connect(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	ctx, cancel := context.WithTimeout(ctx, connectTimeout)
	defer cancel()

	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, internalError("parse postgres dsn", err)
	}
	cfg.ConnConfig.RuntimeParams["timezone"] = databaseZone

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, internalError("create postgres pool", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, internalError("ping postgres", err)
	}

	return pool, nil
}
