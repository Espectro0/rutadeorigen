package createlot

import (
	"rutadeorigen/traceability/internal/application/usecases"
	"rutadeorigen/traceability/internal/domain/lot"
)

type UseCase interface {
	usecases.UseCaseWithReturn[*lot.Lot, *lot.Lot]
}
