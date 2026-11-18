package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"rutadeorigen/qr/internal/application/output/ports"
	"rutadeorigen/qr/internal/domain/stageview"
	"rutadeorigen/qr/internal/infrastructure/output/repositories/structured/postgres/assembler"
	"rutadeorigen/qr/internal/infrastructure/output/repositories/structured/postgres/entity"
	"rutadeorigen/qr/internal/platform/exception"
)

var _ ports.StageViewRepository = (*StageViewRepository)(nil)

const (
	upsertStageView = `
		INSERT INTO lot_stage_view (
			lot_id, stage, event_date, drying_method, drying_days, roast_level,
			roast_duration_minutes, grind, package_weight_grams, best_before
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (lot_id, stage) DO UPDATE SET
			event_date = EXCLUDED.event_date,
			drying_method = EXCLUDED.drying_method,
			drying_days = EXCLUDED.drying_days,
			roast_level = EXCLUDED.roast_level,
			roast_duration_minutes = EXCLUDED.roast_duration_minutes,
			grind = EXCLUDED.grind,
			package_weight_grams = EXCLUDED.package_weight_grams,
			best_before = EXCLUDED.best_before`

	selectStageViewsByLot = `
		SELECT lot_id, stage, event_date, drying_method, drying_days, roast_level,
			roast_duration_minutes, grind, package_weight_grams, best_before
		FROM lot_stage_view
		WHERE lot_id = $1
		ORDER BY CASE stage WHEN 'farm' THEN 1 WHEN 'roasting' THEN 2 ELSE 3 END`
)

type StageViewRepository struct {
	pool      *pgxpool.Pool
	assembler assembler.EntityAssembler[entity.StageView, *stageview.StageView]
}

func NewStageViewRepository(pool *pgxpool.Pool) *StageViewRepository {
	return &StageViewRepository{pool: pool, assembler: assembler.StageView{}}
}

func (r *StageViewRepository) Save(ctx context.Context, view *stageview.StageView) error {
	if view.IsDefault() {
		return exception.New(exception.CodeInternal).WithTechnical("upsert stage view: default view")
	}

	e := r.assembler.ToEntity(view)
	_, err := conn(ctx, r.pool).Exec(ctx, upsertStageView,
		e.LotID, e.Stage, e.EventDate, e.DryingMethod, e.DryingDays, e.RoastLevel,
		e.RoastDurationMinutes, e.Grind, e.PackageWeightGrams, e.BestBefore)
	if err != nil {
		return internalError("upsert stage view", err)
	}
	return nil
}

func (r *StageViewRepository) ListByLotID(ctx context.Context, lotID uuid.UUID) ([]*stageview.StageView, error) {
	rows, err := conn(ctx, r.pool).Query(ctx, selectStageViewsByLot, lotID)
	if err != nil {
		return []*stageview.StageView{}, internalError("query stage views by lot", err)
	}

	entities, err := pgx.CollectRows(rows, pgx.RowToStructByName[entity.StageView])
	if err != nil {
		return []*stageview.StageView{}, internalError("scan stage views", err)
	}

	views := make([]*stageview.StageView, 0, len(entities))
	for _, e := range entities {
		views = append(views, r.assembler.ToDomain(e))
	}
	return views, nil
}
