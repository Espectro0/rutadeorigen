package getlot

import (
	"github.com/google/uuid"

	"rutadeorigen/traceability/internal/application/output/ports"
	"rutadeorigen/traceability/internal/application/usecases"
	"rutadeorigen/traceability/internal/domain/lot"
)

type Result struct {
	Lot            *lot.Lot
	AllowedActions []ports.Action
}

type UseCase interface {
	usecases.UseCaseWithReturn[uuid.UUID, Result]
}
