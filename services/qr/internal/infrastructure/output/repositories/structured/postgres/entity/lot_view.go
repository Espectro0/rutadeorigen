package entity

import (
	"time"

	"github.com/google/uuid"
)

type LotView struct {
	LotID        uuid.UUID `db:"lot_id"`
	Code         string    `db:"code"`
	Variety      string    `db:"variety"`
	Process      string    `db:"process"`
	HarvestDate  time.Time `db:"harvest_date"`
	FarmID       uuid.UUID `db:"farm_id"`
	FarmName     string    `db:"farm_name"`
	FarmAltitude int       `db:"farm_altitude"`
	ProducerID   uuid.UUID `db:"producer_id"`
	ProducerName string    `db:"producer_name"`
	CustodianID  uuid.UUID `db:"custodian_id"`
	CurrentStage string    `db:"current_stage"`
	Status       string    `db:"status"`
	VoidReason   string    `db:"void_reason"`
	Version      int       `db:"version"`
	CreatedAt    time.Time `db:"created_at"`
	UpdatedAt    time.Time `db:"updated_at"`
}
