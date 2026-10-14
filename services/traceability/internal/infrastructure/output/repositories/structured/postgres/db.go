package postgres

import (
	"context"
	_ "embed"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"rutadeorigen/traceability/internal/platform/exception"
)

const connectTimeout = 10 * time.Second

//go:embed schema.sql
var schema string

func Connect(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	ctx, cancel := context.WithTimeout(ctx, connectTimeout)
	defer cancel()

	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, exception.NewInternal(exception.LayerInfrastructure, "", "parse postgres dsn", err)
	}
	cfg.ConnConfig.RuntimeParams["timezone"] = "UTC"

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, exception.NewInternal(exception.LayerInfrastructure, "", "create postgres pool", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, exception.NewInternal(exception.LayerInfrastructure, "", "ping postgres", err)
	}

	if _, err := pool.Exec(ctx, schema); err != nil {
		pool.Close()
		return nil, exception.NewInternal(exception.LayerInfrastructure, "", "apply schema", err)
	}

	return pool, nil
}
