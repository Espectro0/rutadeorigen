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

func ToSummary(a *actor.Actor) response.ActorSummary {
	if a.IsDefault() {
		return response.ActorSummary{}
	}
	return response.ActorSummary{
		ID:   a.ID().String(),
		Type: string(a.Type()),
		Name: a.Name(),
	}
}

func ToList(actors []*actor.Actor) response.ActorList {
	items := make([]response.ActorSummary, 0, len(actors))
	for _, a := range actors {
		if a.IsDefault() {
			continue
		}
		items = append(items, ToSummary(a))
	}
	return response.ActorList{Items: items}
}
