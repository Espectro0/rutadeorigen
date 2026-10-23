package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"rutadeorigen/qr/internal/application/output/ports"
	"rutadeorigen/qr/internal/domain/redirect"
	"rutadeorigen/qr/internal/infrastructure/output/repositories/structured/postgres/assembler"
	"rutadeorigen/qr/internal/infrastructure/output/repositories/structured/postgres/entity"
)

var _ ports.RedirectRepository = (*RedirectRepository)(nil)

const selectActiveRedirects = `
	SELECT id, source_prefix, target_template, status_code, active, created_at
	FROM redirect
	WHERE active
	ORDER BY LENGTH(source_prefix) DESC, source_prefix`

type RedirectRepository struct {
	pool      *pgxpool.Pool
	assembler assembler.EntityAssembler[entity.Redirect, *redirect.Rule]
}

func NewRedirectRepository(pool *pgxpool.Pool) *RedirectRepository {
	return &RedirectRepository{pool: pool, assembler: assembler.Redirect{}}
}

func (r *RedirectRepository) ListActive(ctx context.Context) ([]*redirect.Rule, error) {
	rows, err := conn(ctx, r.pool).Query(ctx, selectActiveRedirects)
	if err != nil {
		return nil, internalError("query active redirects", err)
	}

	entities, err := pgx.CollectRows(rows, pgx.RowToStructByName[entity.Redirect])
	if err != nil {
		return nil, internalError("scan redirects", err)
	}

	rules := make([]*redirect.Rule, 0, len(entities))
	for _, e := range entities {
		rules = append(rules, r.assembler.ToDomain(e))
	}
	return rules, nil
}
