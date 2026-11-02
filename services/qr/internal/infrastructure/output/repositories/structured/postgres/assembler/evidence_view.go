package assembler

import (
	"rutadeorigen/qr/internal/domain/evidenceview"
	"rutadeorigen/qr/internal/domain/lotview"
	"rutadeorigen/qr/internal/infrastructure/output/repositories/structured/postgres/entity"
)

var _ EntityAssembler[entity.EvidenceView, *evidenceview.EvidenceView] = EvidenceView{}

type EvidenceView struct{}

func (EvidenceView) ToDomain(e entity.EvidenceView) *evidenceview.EvidenceView {
	return evidenceview.Restore(evidenceview.RestoreParams{
		EvidenceID:   e.EvidenceID,
		LotID:        e.LotID,
		Stage:        lotview.ParseStage(e.Stage),
		FinalKey:     e.FinalKey,
		ThumbnailKey: e.ThumbnailKey,
		ContentType:  e.ContentType,
		ProcessedAt:  e.ProcessedAt,
	})
}

func (EvidenceView) ToEntity(v *evidenceview.EvidenceView) entity.EvidenceView {
	if v.IsDefault() {
		return entity.EvidenceView{}
	}

	return entity.EvidenceView{
		EvidenceID:   v.EvidenceID(),
		LotID:        v.LotID(),
		Stage:        v.Stage().String(),
		FinalKey:     v.FinalKey(),
		ThumbnailKey: v.ThumbnailKey(),
		ContentType:  v.ContentType(),
		ProcessedAt:  v.ProcessedAt().UTC(),
	}
}
