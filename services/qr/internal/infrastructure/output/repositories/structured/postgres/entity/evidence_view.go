package entity

import (
	"time"

	"github.com/google/uuid"
)

type EvidenceView struct {
	EvidenceID   uuid.UUID `db:"evidence_id"`
	LotID        uuid.UUID `db:"lot_id"`
	Stage        string    `db:"stage"`
	FinalKey     string    `db:"final_key"`
	ThumbnailKey string    `db:"thumbnail_key"`
	ContentType  string    `db:"content_type"`
	ProcessedAt  time.Time `db:"processed_at"`
}
