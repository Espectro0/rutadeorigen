package createlot

import (
	"github.com/google/uuid"
)

type LotCreatedPayload struct {
	LotID       uuid.UUID        `json:"lot_id"`
	Farm        FarmSnapshot     `json:"farm"`
	Producer    ProducerSnapshot `json:"producer"`
	CustodianID uuid.UUID        `json:"custodian_id"`
	Code        string           `json:"code"`
	Variety     string           `json:"variety"`
	Process     string           `json:"process"`
	HarvestDate string           `json:"harvest_date"`
	Stage       string           `json:"stage"`
	Status      string           `json:"status"`
}

type AccessDeniedPayload struct {
	Action       string `json:"action"`
	ResourceType string `json:"resource_type"`
	ResourceID   string `json:"resource_id"`
}

type FarmSnapshot struct {
	ID       uuid.UUID `json:"id"`
	Name     string    `json:"name"`
	Altitude int       `json:"altitude"`
}

type ProducerSnapshot struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}
