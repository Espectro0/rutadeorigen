package transferlot

import "github.com/google/uuid"

type LotTransferredPayload struct {
	LotID       uuid.UUID `json:"lot_id"`
	FromActorID uuid.UUID `json:"from_actor_id"`
	CustodianID uuid.UUID `json:"custodian_id"`
	Stage       string    `json:"stage"`
	Status      string    `json:"status"`
	EventID     uuid.UUID `json:"event_id"`
	Sequence    int       `json:"sequence"`
	EventDate   string    `json:"event_date"`
}
