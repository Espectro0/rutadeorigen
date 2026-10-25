package lotmapper

import (
	"encoding/json"
	"time"

	"rutadeorigen/traceability/internal/application/input/dto/response"
	"rutadeorigen/traceability/internal/domain/lot"
)

func ToEventResponse(e *lot.Event) response.LotEvent {
	if e.IsDefault() {
		return response.LotEvent{}
	}

	details, err := json.Marshal(e.Details())
	if err != nil {
		details = []byte("null")
	}

	return response.LotEvent{
		ID:           e.ID().String(),
		LotID:        e.LotID().String(),
		Sequence:     e.Sequence(),
		Type:         e.Type().String(),
		Stage:        e.Stage().String(),
		EventDate:    e.EventDate().Format(time.DateOnly),
		Note:         e.Note(),
		ActorID:      e.ActorID().String(),
		ActorRole:    e.ActorRole(),
		Details:      details,
		PreviousHash: e.PreviousHash(),
		Hash:         e.Hash(),
		RegisteredAt: e.RegisteredAt().UTC().Format(time.RFC3339Nano),
	}
}
