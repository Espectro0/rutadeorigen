package voidlot

import (
	"github.com/google/uuid"

	"rutadeorigen/traceability/internal/application/usecases"
	"rutadeorigen/traceability/internal/domain/lot"
)

type Command struct {
	LotID           uuid.UUID
	ExpectedVersion int
	Reason          string
}

type UseCase interface {
	usecases.UseCaseWithReturn[Command, *lot.Lot]
}
