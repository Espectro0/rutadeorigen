package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"rutadeorigen/qr/internal/application/output/ports"
	"rutadeorigen/qr/internal/domain/lotview"
	"rutadeorigen/qr/internal/infrastructure/output/repositories/structured/postgres/assembler"
	"rutadeorigen/qr/internal/infrastructure/output/repositories/structured/postgres/entity"
	"rutadeorigen/qr/internal/platform/exception"
)

var _ ports.LotViewRepository = (*LotViewRepository)(nil)

const (
	selectLotView = `
		SELECT lot_id, code, variety, process, harvest_date, farm_id, farm_name, farm_altitude,
			producer_id, producer_name, custodian_id, current_stage, status, void_reason,
			version, created_at, updated_at
		FROM lot_view
		WHERE lot_id = $1`

	upsertLotView = `
		INSERT INTO lot_view (
			lot_id, code, variety, process, harvest_date, farm_id, farm_name, farm_altitude,
			producer_id, producer_name, custodian_id, current_stage, status, void_reason,
			version, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
		ON CONFLICT (lot_id) DO UPDATE SET
			code = EXCLUDED.code,
			variety = EXCLUDED.variety,
			process = EXCLUDED.process,
			harvest_date = EXCLUDED.harvest_date,
			farm_id = EXCLUDED.farm_id,
			farm_name = EXCLUDED.farm_name,
			farm_altitude = EXCLUDED.farm_altitude,
			producer_id = EXCLUDED.producer_id,
			producer_name = EXCLUDED.producer_name,
			custodian_id = EXCLUDED.custodian_id,
			current_stage = EXCLUDED.current_stage,
			status = EXCLUDED.status,
			void_reason = EXCLUDED.void_reason,
			version = EXCLUDED.version,
			updated_at = EXCLUDED.updated_at
		WHERE lot_view.version < EXCLUDED.version`
)

type LotViewRepository struct {
	pool      *pgxpool.Pool
	assembler assembler.EntityAssembler[entity.LotView, *lotview.LotView]
}

func NewLotViewRepository(pool *pgxpool.Pool) *LotViewRepository {
	return &LotViewRepository{pool: pool, assembler: assembler.LotView{}}
}

func (r *LotViewRepository) FindByID(ctx context.Context, id uuid.UUID) (*lotview.LotView, error) {
	rows, err := conn(ctx, r.pool).Query(ctx, selectLotView, id)
	if err != nil {
		return lotview.Default(), internalError("query lot view by id", err)
	}

	row, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[entity.LotView])
	if errors.Is(err, pgx.ErrNoRows) {
		return lotview.Default(), nil
	}
	if err != nil {
		return lotview.Default(), internalError("scan lot view", err)
	}
	return r.assembler.ToDomain(row), nil
}

func (r *LotViewRepository) Save(ctx context.Context, view *lotview.LotView) error {
	if view.IsDefault() {
		return exception.New(exception.CodeInternal).WithTechnical("upsert lot view: default view")
	}

	e := r.assembler.ToEntity(view)
	_, err := conn(ctx, r.pool).Exec(ctx, upsertLotView,
		e.LotID, e.Code, e.Variety, e.Process, e.HarvestDate, e.FarmID, e.FarmName, e.FarmAltitude,
		e.ProducerID, e.ProducerName, e.CustodianID, e.CurrentStage, e.Status, e.VoidReason,
		e.Version, e.CreatedAt, e.UpdatedAt,
	)
	if err != nil {
		return internalError("upsert lot view", err)
	}
	return nil
}
