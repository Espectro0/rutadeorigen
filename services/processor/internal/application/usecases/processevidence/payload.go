package processevidence

import (
	"time"

	"github.com/google/uuid"
)

type EvidenceProcessedPayload struct {
	EvidenceID   uuid.UUID `json:"evidence_id"`
	SHA256       string    `json:"sha256"`
	FinalKey     string    `json:"final_key"`
	ThumbnailKey string    `json:"thumbnail_key"`
	ProcessedAt  time.Time `json:"processed_at"`
}

type EvidenceRejectedPayload struct {
	EvidenceID  uuid.UUID `json:"evidence_id"`
	Reason      string    `json:"reason"`
	ProcessedAt time.Time `json:"processed_at"`
}
