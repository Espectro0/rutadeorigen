package createlot

import (
	"time"

	"github.com/google/uuid"
)

type LotCreatedPayload struct {
	LotID       uuid.UUID `json:"lotId"`
	FarmID      uuid.UUID `json:"farmId"`
	CustodianID uuid.UUID `json:"custodianId"`
	Code        string    `json:"code"`
	Variety     string    `json:"variety"`
	Process     string    `json:"process"`
	HarvestDate time.Time `json:"harvestDate"`
	Stage       string    `json:"stage"`
	Status      string    `json:"status"`
}
