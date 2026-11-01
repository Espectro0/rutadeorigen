package lotmapper

import (
	"encoding/json"
	"time"

	"rutadeorigen/traceability/internal/application/input/dto/response"
	"rutadeorigen/traceability/internal/domain/lot"
	"rutadeorigen/traceability/internal/platform/helper"
)

func ToEventResponse(e *lot.Event) response.LotEvent {
	if e.IsDefault() {
		return response.LotEvent{}
	}

	details := []byte("{}")
	if e.Details() != nil {
		marshaled, err := json.Marshal(e.Details())
		if err != nil {
			marshaled = []byte("null")
		}
		details = marshaled
	}

	target := ""
	if !helper.IsDefaultUUID(e.TargetActorID()) {
		target = e.TargetActorID().String()
	}

	return response.LotEvent{
		ID:            e.ID().String(),
		LotID:         e.LotID().String(),
		Sequence:      e.Sequence(),
		Type:          e.Type().String(),
		Stage:         e.Stage().String(),
		EventDate:     e.EventDate().Format(time.DateOnly),
		Note:          e.Note(),
		ActorID:       e.ActorID().String(),
		ActorRole:     e.ActorRole(),
		TargetActorID: target,
		Details:       details,
		PreviousHash:  e.PreviousHash(),
		Hash:          e.Hash(),
		RegisteredAt:  e.RegisteredAt().UTC().Format(time.RFC3339Nano),
	}
}
