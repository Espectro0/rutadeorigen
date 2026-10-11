package entity

import (
	"time"

	"github.com/google/uuid"
)

type Farm struct {
	ID         uuid.UUID `db:"id"`
	ProducerID uuid.UUID `db:"producer_id"`
	Name       string    `db:"name"`
	Altitude   int       `db:"altitude"`
	CreatedAt  time.Time `db:"created_at"`
}
