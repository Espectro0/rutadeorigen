package postgres

import (
	"context"
	"rutadeorigen/traceability/internal/platform/exception"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"rutadeorigen/traceability/internal/application/output/ports"
	"rutadeorigen/traceability/internal/domain/lot"
	"rutadeorigen/traceability/internal/infrastructure/output/repositories/structured/postgres/assembler"
	"rutadeorigen/traceability/internal/infrastructure/output/repositories/structured/postgres/entity"
)

var _ ports.LotRepository = (*LotRepository)(nil)

const (
	lotFarmCodeConstraint = "lot_farm_code_uk"

	existsLotByCode = `
		SELECT EXISTS (
			SELECT 1 FROM lot WHERE farm_id = $1 AND code = $2
		)`

	insertLot = `
		INSERT INTO lot (
			id, farm_id, custodian_id, code, variety, process,
			harvest_date, current_stage, status, void_reason, version, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`
)

type LotRepository struct {
	pool      *pgxpool.Pool
	assembler assembler.EntityAssembler[entity.Lot, *lot.Lot]
}

func NewLotRepository(pool *pgxpool.Pool) *LotRepository {
	return &LotRepository{pool: pool, assembler: assembler.Lot{}}
}

func (r *LotRepository) ExistsByCode(ctx context.Context, farmID uuid.UUID, code string) (bool, error) {
	var exists bool
	if err := r.pool.QueryRow(ctx, existsLotByCode, farmID, code).Scan(&exists); err != nil {
		return false, internalError("check lot code", err)
	}
	return exists, nil
}

func (r *LotRepository) Save(ctx context.Context, l *lot.Lot) error {
	if l.IsDefault() {
		return exception.New(exception.CodeInternal).
			WithTechnical("insert lot").
			WithCause(ctx.Err())
	}

	e := r.assembler.ToEntity(l)
	_, err := r.pool.Exec(ctx, insertLot,
		e.ID, e.FarmID, e.CustodianID, e.Code, e.Variety, e.Process,
		e.HarvestDate, e.CurrentStage, e.Status, e.VoidReason, e.Version, e.CreatedAt,
	)
	if isUniqueViolation(err, lotFarmCodeConstraint) {
		return exception.New(exception.CodeLotCodeAlreadyExists)
	}
	if err != nil {
		return internalError("insert lot", err)
	}
	return nil
}
