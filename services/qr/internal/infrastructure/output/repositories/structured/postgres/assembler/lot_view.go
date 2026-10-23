package assembler

import (
	"rutadeorigen/qr/internal/domain/lotview"
	"rutadeorigen/qr/internal/infrastructure/output/repositories/structured/postgres/entity"
)

var _ EntityAssembler[entity.LotView, *lotview.LotView] = LotView{}

type LotView struct{}

func (LotView) ToDomain(e entity.LotView) *lotview.LotView {
	return lotview.Restore(lotview.RestoreParams{
		LotID:        e.LotID,
		Code:         e.Code,
		Variety:      e.Variety,
		Process:      lotview.ParseProcess(e.Process),
		HarvestDate:  e.HarvestDate.UTC(),
		FarmID:       e.FarmID,
		FarmName:     e.FarmName,
		FarmAltitude: e.FarmAltitude,
		ProducerID:   e.ProducerID,
		ProducerName: e.ProducerName,
		CustodianID:  e.CustodianID,
		CurrentStage: lotview.ParseStage(e.CurrentStage),
		Status:       lotview.ParseStatus(e.Status),
		VoidReason:   e.VoidReason,
		Version:      e.Version,
		CreatedAt:    e.CreatedAt.UTC(),
		UpdatedAt:    e.UpdatedAt.UTC(),
	})
}

func (LotView) ToEntity(v *lotview.LotView) entity.LotView {
	if v.IsDefault() {
		return entity.LotView{}
	}

	return entity.LotView{
		LotID:        v.LotID(),
		Code:         v.Code(),
		Variety:      v.Variety(),
		Process:      v.Process().String(),
		HarvestDate:  v.HarvestDate().UTC(),
		FarmID:       v.FarmID(),
		FarmName:     v.FarmName(),
		FarmAltitude: v.FarmAltitude(),
		ProducerID:   v.ProducerID(),
		ProducerName: v.ProducerName(),
		CustodianID:  v.CustodianID(),
		CurrentStage: v.CurrentStage().String(),
		Status:       v.Status().String(),
		VoidReason:   v.VoidReason(),
		Version:      v.Version(),
		CreatedAt:    v.CreatedAt().UTC(),
		UpdatedAt:    v.UpdatedAt().UTC(),
	}
}
