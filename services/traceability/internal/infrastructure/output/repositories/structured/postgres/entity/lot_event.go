package entity

import (
	"time"

	"github.com/google/uuid"
)

type LotEvent struct {
	ID           uuid.UUID `db:"id"`
	LotID        uuid.UUID `db:"lot_id"`
	Sequence     int       `db:"sequence"`
	Type         string    `db:"type"`
	Stage        string    `db:"stage"`
	EventDate    time.Time `db:"event_date"`
	Note         string    `db:"note"`
	ActorID      uuid.UUID `db:"actor_id"`
	ActorRole    string    `db:"actor_role"`
	Details      []byte    `db:"details"`
	PreviousHash string    `db:"previous_hash"`
	Hash         string    `db:"hash"`
	RegisteredAt time.Time `db:"registered_at"`
}
