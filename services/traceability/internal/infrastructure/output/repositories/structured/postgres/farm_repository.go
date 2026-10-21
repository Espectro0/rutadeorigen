package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"rutadeorigen/traceability/internal/application/output/ports"
	"rutadeorigen/traceability/internal/domain/farm"
	"rutadeorigen/traceability/internal/infrastructure/output/repositories/structured/postgres/assembler"
	"rutadeorigen/traceability/internal/infrastructure/output/repositories/structured/postgres/entity"
)

var _ ports.FarmRepository = (*FarmRepository)(nil)

const selectFarmByID = `
	SELECT id, producer_id, name, altitude, created_at
	FROM farm
	WHERE id = $1`

type FarmRepository struct {
	pool      *pgxpool.Pool
	assembler assembler.EntityAssembler[entity.Farm, *farm.Farm]
}

func NewFarmRepository(pool *pgxpool.Pool) *FarmRepository {
	return &FarmRepository{pool: pool, assembler: assembler.Farm{}}
}

func (r *FarmRepository) FindByID(ctx context.Context, id uuid.UUID) (*farm.Farm, error) {
	rows, err := conn(ctx, r.pool).Query(ctx, selectFarmByID, id)
	if err != nil {
		return farm.Default(), internalError("query farm by id", err)
	}

	row, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[entity.Farm])
	if errors.Is(err, pgx.ErrNoRows) {
		return farm.Default(), nil
	}
	if err != nil {
		return farm.Default(), internalError("scan farm", err)
	}

	return r.assembler.ToDomain(row), nil
}
