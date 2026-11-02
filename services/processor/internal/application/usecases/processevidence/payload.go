package processevidence

import (
	"time"

	"github.com/google/uuid"
)

type UploaderPayload struct {
	ActorID uuid.UUID `json:"actor_id"`
	Role    string    `json:"role"`
	Name    string    `json:"name"`
	Email   string    `json:"email"`
}

type EvidenceProcessedPayload struct {
	EvidenceID   uuid.UUID `json:"evidence_id"`
	LotID        uuid.UUID `json:"lot_id"`
	LotCode      string    `json:"lot_code"`
	Stage        string    `json:"stage"`
	FileName     string    `json:"file_name"`
	SHA256       string    `json:"sha256"`
	FinalKey     string    `json:"final_key"`
	ThumbnailKey string    `json:"thumbnail_key"`
	ContentType  string    `json:"content_type"`
	ProcessedAt  time.Time `json:"processed_at"`

	Uploader UploaderPayload `json:"uploader"`
}

type EvidenceRejectedPayload struct {
	EvidenceID  uuid.UUID `json:"evidence_id"`
	LotID       uuid.UUID `json:"lot_id"`
	LotCode     string    `json:"lot_code"`
	Stage       string    `json:"stage"`
	FileName    string    `json:"file_name"`
	Reason      string    `json:"reason"`
	ProcessedAt time.Time `json:"processed_at"`

	Uploader UploaderPayload `json:"uploader"`
}
