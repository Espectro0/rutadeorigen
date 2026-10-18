package lotmapper

import (
	"time"

	"rutadeorigen/traceability/internal/application/input/dto/response"
	"rutadeorigen/traceability/internal/domain/lot"
)

func ToResponse(l *lot.Lot) response.Lot {
	if l.IsDefault() {
		return response.Lot{}
	}

	return response.Lot{
		ID:          l.ID().String(),
		Version:     l.Version(),
		FarmID:      l.FarmID().String(),
		CustodianID: l.CustodianID().String(),
		Code:        l.Code(),
		Variety:     l.Variety(),
		Process:     l.Process().String(),
		HarvestDate: l.HarvestDate().Format(time.DateOnly),
		Stage:       l.CurrentStage().String(),
		Status:      l.Status().String(),
		VoidReason:  l.VoidReason(),
		CreatedAt:   l.CreatedAt().Format(time.RFC3339),
	}
}
