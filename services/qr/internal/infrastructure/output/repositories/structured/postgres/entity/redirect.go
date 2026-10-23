package entity

import (
	"time"

	"github.com/google/uuid"
)

type Redirect struct {
	ID             uuid.UUID `db:"id"`
	SourcePrefix   string    `db:"source_prefix"`
	TargetTemplate string    `db:"target_template"`
	StatusCode     int       `db:"status_code"`
	Active         bool      `db:"active"`
	CreatedAt      time.Time `db:"created_at"`
}
