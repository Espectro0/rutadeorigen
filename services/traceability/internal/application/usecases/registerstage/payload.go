package registerstage

import (
	"github.com/google/uuid"

	"rutadeorigen/traceability/internal/domain/lot"
)

type StageRegisteredPayload struct {
	LotID     uuid.UUID `json:"lot_id"`
	Stage     string    `json:"stage"`
	Status    string    `json:"status"`
	EventID   uuid.UUID `json:"event_id"`
	Sequence  int       `json:"sequence"`
	EventDate string    `json:"event_date"`

	Details lot.StageDetails `json:"details"`
}
