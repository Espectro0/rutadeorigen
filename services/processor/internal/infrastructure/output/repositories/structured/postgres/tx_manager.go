package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"rutadeorigen/processor/internal/application/output/ports"
	"rutadeorigen/processor/internal/platform/exception"
)

var _ ports.TxManager = (*TxManager)(nil)

type txKey struct{}

type executor interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type TxManager struct{ pool *pgxpool.Pool }

func NewTxManager(pool *pgxpool.Pool) *TxManager { return &TxManager{pool: pool} }

func (m *TxManager) WithinTx(ctx context.Context, fn func(context.Context) error) error {
	if _, insideTx := ctx.Value(txKey{}).(pgx.Tx); insideTx {
		return fn(ctx)
	}

	err := pgx.BeginFunc(ctx, m.pool, func(tx pgx.Tx) error {
		return fn(context.WithValue(ctx, txKey{}, tx))
	})
	return exception.Wrap(err, "run transaction")
}

func conn(ctx context.Context, pool *pgxpool.Pool) executor {
	if tx, insideTx := ctx.Value(txKey{}).(pgx.Tx); insideTx {
		return tx
	}
	return pool
}
