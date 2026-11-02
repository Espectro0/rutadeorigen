package processevidence

import (
	"time"

	"github.com/google/uuid"
)

type EvidenceProcessedPayload struct {
	EvidenceID   uuid.UUID `json:"evidence_id"`
	LotID        uuid.UUID `json:"lot_id"`
	Stage        string    `json:"stage"`
	SHA256       string    `json:"sha256"`
	FinalKey     string    `json:"final_key"`
	ThumbnailKey string    `json:"thumbnail_key"`
	ContentType  string    `json:"content_type"`
	ProcessedAt  time.Time `json:"processed_at"`
}

type EvidenceRejectedPayload struct {
	EvidenceID  uuid.UUID `json:"evidence_id"`
	Reason      string    `json:"reason"`
	ProcessedAt time.Time `json:"processed_at"`
}
