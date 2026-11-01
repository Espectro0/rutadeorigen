package transferlot

import (
	"github.com/google/uuid"

	"rutadeorigen/traceability/internal/application/usecases"
	"rutadeorigen/traceability/internal/domain/lot"
)

type Command struct {
	LotID           uuid.UUID
	ExpectedVersion int
	TargetActorID   uuid.UUID
	Note            string
}

type Result struct {
	Event      *lot.Event
	LotVersion int
}

type UseCase interface {
	usecases.UseCaseWithReturn[Command, Result]
}
