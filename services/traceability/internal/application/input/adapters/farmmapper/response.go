package farmmapper

import (
	"time"

	"rutadeorigen/traceability/internal/application/input/dto/response"
	"rutadeorigen/traceability/internal/domain/farm"
)

func ToResponse(f *farm.Farm) response.Farm {
	if f.IsDefault() {
		return response.Farm{}
	}

	return response.Farm{
		ID:         f.ID().String(),
		ProducerID: f.ProducerID().String(),
		Name:       f.Name(),
		Altitude:   f.Altitude(),
		CreatedAt:  f.CreatedAt().Format(time.RFC3339),
	}
}

func ToList(farms []*farm.Farm) response.FarmList {
	items := make([]response.Farm, 0, len(farms))
	for _, f := range farms {
		if f.IsDefault() {
			continue
		}
		items = append(items, ToResponse(f))
	}
	return response.FarmList{Items: items}
}
