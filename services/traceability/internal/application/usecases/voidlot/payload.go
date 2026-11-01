package voidlot

import "github.com/google/uuid"

type LotVoidedPayload struct {
	LotID       uuid.UUID `json:"lot_id"`
	VoidedBy    uuid.UUID `json:"voided_by"`
	Reason      string    `json:"reason"`
	CustodianID uuid.UUID `json:"custodian_id"`
	Stage       string    `json:"stage"`
	Status      string    `json:"status"`
}
