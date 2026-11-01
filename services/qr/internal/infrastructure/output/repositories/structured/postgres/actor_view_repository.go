package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"rutadeorigen/qr/internal/application/output/ports"
	"rutadeorigen/qr/internal/domain/actorview"
	"rutadeorigen/qr/internal/infrastructure/output/repositories/structured/postgres/assembler"
	"rutadeorigen/qr/internal/infrastructure/output/repositories/structured/postgres/entity"
	"rutadeorigen/qr/internal/platform/exception"
)

var _ ports.ActorViewRepository = (*ActorViewRepository)(nil)

const (
	selectActorViewByUser = `
		SELECT actor_id, user_id, role, created_at
		FROM actor_view
		WHERE user_id = $1`

	upsertActorView = `
		INSERT INTO actor_view (actor_id, user_id, role, created_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (actor_id) DO UPDATE SET role = EXCLUDED.role`
)

type ActorViewRepository struct {
	pool      *pgxpool.Pool
	assembler assembler.EntityAssembler[entity.ActorView, *actorview.ActorView]
}

func NewActorViewRepository(pool *pgxpool.Pool) *ActorViewRepository {
	return &ActorViewRepository{pool: pool, assembler: assembler.ActorView{}}
}

func (r *ActorViewRepository) FindByUserID(ctx context.Context, userID string) (*actorview.ActorView, error) {
	rows, err := conn(ctx, r.pool).Query(ctx, selectActorViewByUser, userID)
	if err != nil {
		return actorview.Default(), internalError("query actor view by user id", err)
	}

	row, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[entity.ActorView])
	if errors.Is(err, pgx.ErrNoRows) {
		return actorview.Default(), nil
	}
	if err != nil {
		return actorview.Default(), internalError("scan actor view", err)
	}
	return r.assembler.ToDomain(row), nil
}

func (r *ActorViewRepository) Save(ctx context.Context, view *actorview.ActorView) error {
	if view.IsDefault() {
		return exception.New(exception.CodeInternal).WithTechnical("upsert actor view: default view")
	}

	e := r.assembler.ToEntity(view)
	if _, err := conn(ctx, r.pool).Exec(ctx, upsertActorView, e.ActorID, e.UserID, e.Role, e.CreatedAt); err != nil {
		return internalError("upsert actor view", err)
	}
	return nil
}
