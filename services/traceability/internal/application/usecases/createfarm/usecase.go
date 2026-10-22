package createfarm

import (
	"rutadeorigen/traceability/internal/application/usecases"
	"rutadeorigen/traceability/internal/domain/farm"
)

type UseCase interface {
	usecases.UseCaseWithoutReturn[*farm.Farm]
}
