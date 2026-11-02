package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"rutadeorigen/qr/internal/application/output/ports"
	"rutadeorigen/qr/internal/domain/evidenceview"
	"rutadeorigen/qr/internal/infrastructure/output/repositories/structured/postgres/assembler"
	"rutadeorigen/qr/internal/infrastructure/output/repositories/structured/postgres/entity"
	"rutadeorigen/qr/internal/platform/exception"
)

var _ ports.EvidenceViewRepository = (*EvidenceViewRepository)(nil)

const (
	insertEvidenceView = `
		INSERT INTO evidence_view (evidence_id, lot_id, stage, final_key, thumbnail_key, content_type, processed_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (evidence_id) DO NOTHING`

	selectEvidenceViewsByLot = `
		SELECT evidence_id, lot_id, stage, final_key, thumbnail_key, content_type, processed_at
		FROM evidence_view
		WHERE lot_id = $1
		ORDER BY processed_at, evidence_id
		LIMIT $2`
)

type EvidenceViewRepository struct {
	pool      *pgxpool.Pool
	assembler assembler.EntityAssembler[entity.EvidenceView, *evidenceview.EvidenceView]
}

func NewEvidenceViewRepository(pool *pgxpool.Pool) *EvidenceViewRepository {
	return &EvidenceViewRepository{pool: pool, assembler: assembler.EvidenceView{}}
}

func (r *EvidenceViewRepository) Save(ctx context.Context, view *evidenceview.EvidenceView) error {
	if view.IsDefault() {
		return exception.New(exception.CodeInternal).WithTechnical("insert evidence view: default view")
	}

	e := r.assembler.ToEntity(view)
	_, err := conn(ctx, r.pool).Exec(ctx, insertEvidenceView,
		e.EvidenceID, e.LotID, e.Stage, e.FinalKey, e.ThumbnailKey, e.ContentType, e.ProcessedAt)
	if err != nil {
		return internalError("insert evidence view", err)
	}
	return nil
}

func (r *EvidenceViewRepository) ListByLotID(ctx context.Context, lotID uuid.UUID, limit int) ([]*evidenceview.EvidenceView, error) {
	rows, err := conn(ctx, r.pool).Query(ctx, selectEvidenceViewsByLot, lotID, limit)
	if err != nil {
		return []*evidenceview.EvidenceView{}, internalError("query evidence views by lot", err)
	}

	entities, err := pgx.CollectRows(rows, pgx.RowToStructByName[entity.EvidenceView])
	if err != nil {
		return []*evidenceview.EvidenceView{}, internalError("scan evidence views", err)
	}

	views := make([]*evidenceview.EvidenceView, 0, len(entities))
	for _, e := range entities {
		views = append(views, r.assembler.ToDomain(e))
	}
	return views, nil
}
