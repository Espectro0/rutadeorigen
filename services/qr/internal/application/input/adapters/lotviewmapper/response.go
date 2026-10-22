package lotviewmapper

import (
	"time"

	"rutadeorigen/qr/internal/application/input/dto/response"
	"rutadeorigen/qr/internal/domain/lotview"
)

func ToPublicLot(v *lotview.LotView) response.PublicLot {
	if v.IsDefault() {
		return response.PublicLot{}
	}

	return response.PublicLot{
		ID:          v.LotID().String(),
		Code:        v.Code(),
		Variety:     v.Variety(),
		Process:     v.Process().String(),
		HarvestDate: v.HarvestDate().Format(time.DateOnly),
		Farm: response.PublicFarm{
			Name:     v.FarmName(),
			Altitude: v.FarmAltitude(),
		},
		Producer: response.PublicProducer{
			Name: v.ProducerName(),
		},
		Stage:       v.CurrentStage().String(),
		Status:      v.Status().String(),
		VoidReason:  v.VoidReason(),
		QRAvailable: v.CanGenerateImage() == nil,
		UpdatedAt:   v.UpdatedAt().Format(time.RFC3339),
	}
}
