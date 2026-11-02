package completeevidenceupload

import (
	"github.com/google/uuid"

	"rutadeorigen/traceability/internal/application/usecases"
	"rutadeorigen/traceability/internal/domain/evidence"
)

type Command struct {
	LotID      uuid.UUID
	EvidenceID uuid.UUID
}

type UseCase interface {
	usecases.UseCaseWithReturn[Command, *evidence.Evidence]
}
