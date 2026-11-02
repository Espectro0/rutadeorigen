package assembler

import (
	"time"

	"rutadeorigen/traceability/internal/domain/evidence"
	"rutadeorigen/traceability/internal/domain/lot"
	"rutadeorigen/traceability/internal/infrastructure/output/repositories/structured/postgres/entity"
	"rutadeorigen/traceability/internal/platform/helper"
)

var _ EntityAssembler[entity.Evidence, *evidence.Evidence] = Evidence{}

type Evidence struct{}

func (Evidence) ToDomain(e entity.Evidence) *evidence.Evidence {
	processedAt := helper.DefaultDate()
	if e.ProcessedAt != nil {
		processedAt = e.ProcessedAt.UTC()
	}

	return evidence.Restore(evidence.RestoreParams{
		ID:              e.ID,
		LotID:           e.LotID,
		Stage:           lot.ParseStage(e.Stage),
		UploadedBy:      e.UploadedBy,
		FileName:        e.FileName,
		ContentType:     evidence.ParseContentType(e.ContentType),
		SizeBytes:       e.SizeBytes,
		Status:          evidence.ParseStatus(e.Status),
		SHA256:          e.SHA256,
		FinalKey:        e.FinalKey,
		ThumbnailKey:    e.ThumbnailKey,
		RejectionReason: e.RejectionReason,
		CreatedAt:       e.CreatedAt.UTC(),
		ProcessedAt:     processedAt,
	})
}

func (Evidence) ToEntity(d *evidence.Evidence) entity.Evidence {
	if d.IsDefault() {
		return entity.Evidence{}
	}

	var processedAt *time.Time
	if !helper.IsDefaultDate(d.ProcessedAt()) {
		at := d.ProcessedAt().UTC()
		processedAt = &at
	}

	return entity.Evidence{
		ID:              d.ID(),
		LotID:           d.LotID(),
		Stage:           d.Stage().String(),
		UploadedBy:      d.UploadedBy(),
		FileName:        d.FileName(),
		ContentType:     d.ContentType().String(),
		SizeBytes:       d.SizeBytes(),
		Status:          d.Status().String(),
		SHA256:          d.SHA256(),
		FinalKey:        d.FinalKey(),
		ThumbnailKey:    d.ThumbnailKey(),
		RejectionReason: d.RejectionReason(),
		CreatedAt:       d.CreatedAt().UTC(),
		ProcessedAt:     processedAt,
	}
}
