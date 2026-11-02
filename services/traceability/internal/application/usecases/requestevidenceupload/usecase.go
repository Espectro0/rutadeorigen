package requestevidenceupload

import (
	"github.com/google/uuid"

	"rutadeorigen/traceability/internal/application/output/ports"
	"rutadeorigen/traceability/internal/application/usecases"
	"rutadeorigen/traceability/internal/domain/evidence"
)

type Command struct {
	LotID       uuid.UUID
	FileName    string
	ContentType string
	SizeBytes   int64
}

type Result struct {
	Evidence *evidence.Evidence
	Upload   ports.PresignedUpload
}

type UseCase interface {
	usecases.UseCaseWithReturn[Command, Result]
}
