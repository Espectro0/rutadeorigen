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
	"rutadeorigen/traceability/internal/platform/exception"
)

var _ ports.ActorRepository = (*ActorRepository)(nil)

const (
	actorUserIDConstraint = "actor_user_id_key"

	selectActorByID = `
	SELECT id, user_id, type, name, email, status, created_at
	FROM actor
	WHERE id = $1`

	selectActorByUserID = `
	SELECT id, user_id, type, name, email, status, created_at
	FROM actor
	WHERE user_id = $1`

	upsertActor = `
	INSERT INTO actor (id, user_id, type, name, email, status, created_at)
	VALUES ($1, $2, $3, $4, $5, $6, $7)
	ON CONFLICT (id) DO UPDATE
	SET type = EXCLUDED.type, name = EXCLUDED.name, email = EXCLUDED.email, status = EXCLUDED.status`
)

type ActorRepository struct {
	pool      *pgxpool.Pool
	assembler assembler.EntityAssembler[entity.Actor, *actor.Actor]
}

func NewActorRepository(pool *pgxpool.Pool) *ActorRepository {
	return &ActorRepository{pool: pool, assembler: assembler.Actor{}}
}

func (r *ActorRepository) FindByID(ctx context.Context, id uuid.UUID) (*actor.Actor, error) {
	return r.findOne(ctx, selectActorByID, id, "actor by id")
}

func (r *ActorRepository) FindByUserID(ctx context.Context, userID string) (*actor.Actor, error) {
	return r.findOne(ctx, selectActorByUserID, userID, "actor by user id")
}

func (r *ActorRepository) Save(ctx context.Context, a *actor.Actor) error {
	if a.IsDefault() {
		return exception.New(exception.CodeInternal).WithTechnical("cannot save a default actor")
	}

	e := r.assembler.ToEntity(a)
	_, err := conn(ctx, r.pool).Exec(ctx, upsertActor,
		e.ID, e.UserID, e.Type, e.Name, e.Email, e.Status, e.CreatedAt,
	)
	if isUniqueViolation(err, actorUserIDConstraint) {
		return exception.New(exception.CodeActorAlreadyExists)
	}
	if err != nil {
		return internalError("save actor", err)
	}
	return nil
}

func (r *ActorRepository) findOne(ctx context.Context, query string, arg any, what string) (*actor.Actor, error) {
	rows, err := conn(ctx, r.pool).Query(ctx, query, arg)
	if err != nil {
		return actor.Default(), internalError("query "+what, err)
	}

	row, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[entity.Actor])
	if errors.Is(err, pgx.ErrNoRows) {
		return actor.Default(), nil
	}
	if err != nil {
		return actor.Default(), internalError("scan "+what, err)
	}

	return r.assembler.ToDomain(row), nil
}
