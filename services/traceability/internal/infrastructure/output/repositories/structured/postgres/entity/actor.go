package entity

import (
	"time"

	"github.com/google/uuid"
)

type Actor struct {
	ID        uuid.UUID `db:"id"`
	UserID    string    `db:"user_id"`
	Type      string    `db:"type"`
	Name      string    `db:"name"`
	Status    string    `db:"status"`
	CreatedAt time.Time `db:"created_at"`
}
