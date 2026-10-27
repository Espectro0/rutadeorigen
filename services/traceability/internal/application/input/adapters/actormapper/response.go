package actormapper

import (
	"time"

	"rutadeorigen/traceability/internal/application/input/dto/response"
	"rutadeorigen/traceability/internal/domain/actor"
)

func ToResponse(a *actor.Actor) response.Actor {
	if a.IsDefault() {
		return response.Actor{}
	}
	return response.Actor{
		ID:        a.ID().String(),
		Type:      string(a.Type()),
		Name:      a.Name(),
		Email:     a.Email(),
		Status:    string(a.Status()),
		CreatedAt: a.CreatedAt().Format(time.RFC3339),
	}
}
