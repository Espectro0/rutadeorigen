package entity

import (
	"time"

	"github.com/google/uuid"
)

type Evidence struct {
	ID              uuid.UUID  `db:"id"`
	LotID           uuid.UUID  `db:"lot_id"`
	Stage           string     `db:"stage"`
	UploadedBy      uuid.UUID  `db:"uploaded_by"`
	FileName        string     `db:"file_name"`
	ContentType     string     `db:"content_type"`
	SizeBytes       int64      `db:"size_bytes"`
	Status          string     `db:"status"`
	SHA256          string     `db:"sha256"`
	FinalKey        string     `db:"final_key"`
	ThumbnailKey    string     `db:"thumbnail_key"`
	RejectionReason string     `db:"rejection_reason"`
	CreatedAt       time.Time  `db:"created_at"`
	ProcessedAt     *time.Time `db:"processed_at"`
}
