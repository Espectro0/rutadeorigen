package registerstage

import (
	"time"

	"github.com/google/uuid"

	"rutadeorigen/traceability/internal/application/usecases"
	"rutadeorigen/traceability/internal/domain/lot"
)

type Command struct {
	LotID           uuid.UUID
	ExpectedVersion int
	Stage           lot.Stage
	EventDate       time.Time
	Note            string
	Details         lot.StageDetails
}

type Result struct {
	Event      *lot.Event
	LotVersion int
}

type UseCase interface {
	usecases.UseCaseWithReturn[Command, Result]
}
