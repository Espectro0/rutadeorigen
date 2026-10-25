package response

import "encoding/json"

type LotEvent struct {
	ID           string          `json:"id"`
	LotID        string          `json:"lot_id"`
	Sequence     int             `json:"sequence"`
	Type         string          `json:"type"`
	Stage        string          `json:"stage"`
	EventDate    string          `json:"event_date"`
	Note         string          `json:"note,omitempty"`
	ActorID      string          `json:"actor_id"`
	ActorRole    string          `json:"actor_role"`
	Details      json.RawMessage `json:"details"`
	PreviousHash string          `json:"previous_hash"`
	Hash         string          `json:"hash"`
	RegisteredAt string          `json:"registered_at"`
}

type RegisteredStage struct {
	Event      LotEvent `json:"event"`
	LotVersion int      `json:"lot_version"`
}
