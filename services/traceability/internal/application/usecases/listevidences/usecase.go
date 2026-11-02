package listevidences

import (
	"github.com/google/uuid"

	"rutadeorigen/traceability/internal/application/usecases"
	"rutadeorigen/traceability/internal/domain/evidence"
)

type UseCase interface {
	usecases.UseCaseWithReturn[uuid.UUID, []*evidence.Evidence]
}
