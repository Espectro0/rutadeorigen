package deletefarm

import (
	"github.com/google/uuid"

	"rutadeorigen/traceability/internal/application/usecases"
)

type UseCase interface {
	usecases.UseCaseWithoutReturn[uuid.UUID]
}
