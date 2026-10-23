package entity

import (
	"time"

	"github.com/google/uuid"
)

type QRCode struct {
	LotID           uuid.UUID `db:"lot_id"`
	Content         string    `db:"content"`
	ErrorCorrection string    `db:"error_correction"`
	LibraryVersion  string    `db:"library_version"`
	CreatedAt       time.Time `db:"created_at"`
}
