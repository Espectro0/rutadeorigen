package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"rutadeorigen/traceability/internal/application/output/ports"
	"rutadeorigen/traceability/internal/domain/evidence"
	"rutadeorigen/traceability/internal/infrastructure/output/repositories/structured/postgres/assembler"
	"rutadeorigen/traceability/internal/infrastructure/output/repositories/structured/postgres/entity"
	"rutadeorigen/traceability/internal/platform/exception"
)

var _ ports.EvidenceRepository = (*EvidenceRepository)(nil)

const (
	selectEvidence = `
		SELECT id, lot_id, stage, uploaded_by, file_name, content_type, size_bytes, status,
		       sha256, final_key, thumbnail_key, rejection_reason, created_at, processed_at
		FROM evidence`

	selectEvidenceByID = selectEvidence + `
		WHERE id = $1`

	selectEvidencesByLot = selectEvidence + `
		WHERE lot_id = $1
		ORDER BY created_at DESC, id DESC`

	insertEvidence = `
		INSERT INTO evidence (
			id, lot_id, stage, uploaded_by, file_name, content_type, size_bytes, status,
			sha256, final_key, thumbnail_key, rejection_reason, created_at, processed_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`

	updateEvidence = `
		UPDATE evidence
		SET status = $2, sha256 = $3, final_key = $4, thumbnail_key = $5,
		    rejection_reason = $6, processed_at = $7
		WHERE id = $1`
)

type EvidenceRepository struct {
	pool      *pgxpool.Pool
	assembler assembler.EntityAssembler[entity.Evidence, *evidence.Evidence]
}

func NewEvidenceRepository(pool *pgxpool.Pool) *EvidenceRepository {
	return &EvidenceRepository{pool: pool, assembler: assembler.Evidence{}}
}

func (r *EvidenceRepository) Save(ctx context.Context, e *evidence.Evidence) error {
	if e.IsDefault() {
		return exception.New(exception.CodeInternal).WithTechnical("cannot save a default evidence")
	}

	row := r.assembler.ToEntity(e)
	if _, err := conn(ctx, r.pool).Exec(ctx, insertEvidence,
		row.ID, row.LotID, row.Stage, row.UploadedBy, row.FileName, row.ContentType, row.SizeBytes, row.Status,
		row.SHA256, row.FinalKey, row.ThumbnailKey, row.RejectionReason, row.CreatedAt, row.ProcessedAt,
	); err != nil {
		return internalError("insert evidence", err)
	}
	return nil
}

func (r *EvidenceRepository) Update(ctx context.Context, e *evidence.Evidence) error {
	if e.IsDefault() {
		return exception.New(exception.CodeInternal).WithTechnical("cannot update a default evidence")
	}

	row := r.assembler.ToEntity(e)
	tag, err := conn(ctx, r.pool).Exec(ctx, updateEvidence,
		row.ID, row.Status, row.SHA256, row.FinalKey, row.ThumbnailKey, row.RejectionReason, row.ProcessedAt)
	if err != nil {
		return internalError("update evidence", err)
	}
	if tag.RowsAffected() == 0 {
		return exception.New(exception.CodeEvidenceNotFound)
	}
	return nil
}

func (r *EvidenceRepository) FindByID(ctx context.Context, id uuid.UUID) (*evidence.Evidence, error) {
	rows, err := conn(ctx, r.pool).Query(ctx, selectEvidenceByID, id)
	if err != nil {
		return evidence.Default(), internalError("query evidence by id", err)
	}

	row, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[entity.Evidence])
	if errors.Is(err, pgx.ErrNoRows) {
		return evidence.Default(), nil
	}
	if err != nil {
		return evidence.Default(), internalError("scan evidence", err)
	}

	return r.assembler.ToDomain(row), nil
}

func (r *EvidenceRepository) ListByLotID(ctx context.Context, lotID uuid.UUID) ([]*evidence.Evidence, error) {
	rows, err := conn(ctx, r.pool).Query(ctx, selectEvidencesByLot, lotID)
	if err != nil {
		return []*evidence.Evidence{}, internalError("query evidences", err)
	}

	entities, err := pgx.CollectRows(rows, pgx.RowToStructByName[entity.Evidence])
	if err != nil {
		return []*evidence.Evidence{}, internalError("scan evidences", err)
	}

	items := make([]*evidence.Evidence, 0, len(entities))
	for _, row := range entities {
		items = append(items, r.assembler.ToDomain(row))
	}
	return items, nil
}
