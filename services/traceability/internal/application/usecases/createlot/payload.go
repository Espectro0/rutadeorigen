package createlot

import (
	"github.com/google/uuid"
)

type LotCreatedPayload struct {
	LotID       uuid.UUID `json:"lot_id"`
	FarmID      uuid.UUID `json:"farm_id"`
	CustodianID uuid.UUID `json:"custodian_id"`
	Code        string    `json:"code"`
	Variety     string    `json:"variety"`
	Process     string    `json:"process"`
	HarvestDate string    `json:"harvest_date"`
	Stage       string    `json:"stage"`
	Status      string    `json:"status"`
}
