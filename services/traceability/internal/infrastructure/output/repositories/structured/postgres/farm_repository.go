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
	"rutadeorigen/traceability/internal/platform/exception"
)

var _ ports.FarmRepository = (*FarmRepository)(nil)

const (
	selectFarmByID = `
		SELECT id, producer_id, name, altitude, created_at
		FROM farm
		WHERE id = $1`

	selectFarmsByProducer = `
		SELECT id, producer_id, name, altitude, created_at
		FROM farm
		WHERE producer_id = $1
		ORDER BY created_at DESC, id DESC`

	insertFarm = `
		INSERT INTO farm (id, producer_id, name, altitude, created_at)
		VALUES ($1, $2, $3, $4, $5)`
)

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

func (r *FarmRepository) Save(ctx context.Context, f *farm.Farm) error {
	if f.IsDefault() {
		return exception.New(exception.CodeInternal).WithTechnical("cannot save a default farm")
	}

	e := r.assembler.ToEntity(f)
	if _, err := conn(ctx, r.pool).Exec(ctx, insertFarm,
		e.ID, e.ProducerID, e.Name, e.Altitude, e.CreatedAt,
	); err != nil {
		return internalError("insert farm", err)
	}
	return nil
}

func (r *FarmRepository) ListByProducer(ctx context.Context, producerID uuid.UUID) ([]*farm.Farm, error) {
	rows, err := conn(ctx, r.pool).Query(ctx, selectFarmsByProducer, producerID)
	if err != nil {
		return []*farm.Farm{}, internalError("query farms by producer", err)
	}

	entities, err := pgx.CollectRows(rows, pgx.RowToStructByName[entity.Farm])
	if err != nil {
		return []*farm.Farm{}, internalError("scan farms", err)
	}

	farms := make([]*farm.Farm, 0, len(entities))
	for _, e := range entities {
		farms = append(farms, r.assembler.ToDomain(e))
	}
	return farms, nil
}
