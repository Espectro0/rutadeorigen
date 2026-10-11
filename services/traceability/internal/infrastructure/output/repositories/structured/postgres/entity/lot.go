package entity

import (
	"time"

	"github.com/google/uuid"
)

type Lot struct {
	ID           uuid.UUID `db:"id"`
	FarmID       uuid.UUID `db:"farm_id"`
	CustodianID  uuid.UUID `db:"custodian_id"`
	Code         string    `db:"code"`
	Variety      string    `db:"variety"`
	Process      string    `db:"process"`
	HarvestDate  time.Time `db:"harvest_date"`
	CurrentStage string    `db:"current_stage"`
	Status       string    `db:"status"`
	VoidReason   string    `db:"void_reason"`
	Version      int       `db:"version"`
	CreatedAt    time.Time `db:"created_at"`
}
