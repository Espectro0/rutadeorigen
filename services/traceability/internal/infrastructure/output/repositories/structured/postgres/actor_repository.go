package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"rutadeorigen/traceability/internal/application/output/ports"
	"rutadeorigen/traceability/internal/domain/actor"
	"rutadeorigen/traceability/internal/infrastructure/output/repositories/structured/postgres/assembler"
	"rutadeorigen/traceability/internal/infrastructure/output/repositories/structured/postgres/entity"
)

var _ ports.ActorRepository = (*ActorRepository)(nil)

const selectActorByID = `
	SELECT id, user_id, type, name, status, created_at
	FROM actor
	WHERE id = $1`

type ActorRepository struct {
	pool      *pgxpool.Pool
	assembler assembler.EntityAssembler[entity.Actor, *actor.Actor]
}

func NewActorRepository(pool *pgxpool.Pool) *ActorRepository {
	return &ActorRepository{pool: pool, assembler: assembler.Actor{}}
}

func (r *ActorRepository) FindByID(ctx context.Context, id uuid.UUID) (*actor.Actor, error) {
	rows, err := conn(ctx, r.pool).Query(ctx, selectActorByID, id)
	if err != nil {
		return actor.Default(), internalError("query actor by id", err)
	}

	row, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[entity.Actor])
	if errors.Is(err, pgx.ErrNoRows) {
		return actor.Default(), nil
	}
	if err != nil {
		return actor.Default(), internalError("scan actor", err)
	}

	return r.assembler.ToDomain(row), nil
}
