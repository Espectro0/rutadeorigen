package getlot

import (
	"github.com/google/uuid"

	"rutadeorigen/traceability/internal/application/usecases"
	"rutadeorigen/traceability/internal/domain/lot"
)

type UseCase interface {
	usecases.UseCaseWithReturn[uuid.UUID, *lot.Lot]
}
