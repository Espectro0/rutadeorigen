package entity

import (
	"time"

	"github.com/google/uuid"
)

type ActorView struct {
	ActorID   uuid.UUID `db:"actor_id"`
	UserID    string    `db:"user_id"`
	Role      string    `db:"role"`
	CreatedAt time.Time `db:"created_at"`
}
