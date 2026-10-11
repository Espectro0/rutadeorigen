package assembler

import (
	"rutadeorigen/traceability/internal/domain/lot"
	"rutadeorigen/traceability/internal/infrastructure/output/repositories/structured/postgres/entity"
)

var _ EntityAssembler[entity.Lot, *lot.Lot] = Lot{}

type Lot struct{}

func (Lot) ToDomain(e entity.Lot) *lot.Lot {
	return lot.Restore(lot.RestoreParams{
		ID:           e.ID,
		FarmID:       e.FarmID,
		CustodianID:  e.CustodianID,
		Code:         e.Code,
		Variety:      e.Variety,
		Process:      lot.ParseProcess(e.Process),
		HarvestDate:  e.HarvestDate,
		CurrentStage: lot.ParseStage(e.CurrentStage),
		Status:       lot.ParseStatus(e.Status),
		VoidReason:   e.VoidReason,
		Version:      e.Version,
		CreatedAt:    e.CreatedAt,
	})
}

func (Lot) ToEntity(l *lot.Lot) entity.Lot {
	if l.IsDefault() {
		return entity.Lot{}
	}

	return entity.Lot{
		ID:           l.ID(),
		FarmID:       l.FarmID(),
		CustodianID:  l.CustodianID(),
		Code:         l.Code(),
		Variety:      l.Variety(),
		Process:      l.Process().String(),
		HarvestDate:  l.HarvestDate(),
		CurrentStage: l.CurrentStage().String(),
		Status:       l.Status().String(),
		VoidReason:   l.VoidReason(),
		Version:      l.Version(),
		CreatedAt:    l.CreatedAt(),
	}
}
