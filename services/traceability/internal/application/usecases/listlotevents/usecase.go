package listlotevents

import (
	"github.com/google/uuid"

	"rutadeorigen/traceability/internal/application/usecases"
	"rutadeorigen/traceability/internal/domain/lot"
)

type Result struct {
	Events     []*lot.Event
	ChainValid bool
}

type UseCase interface {
	usecases.UseCaseWithReturn[uuid.UUID, Result]
}
