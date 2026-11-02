package processevidence

import (
	"github.com/google/uuid"

	"rutadeorigen/processor/internal/application/usecases"
)

type Uploader struct {
	ActorID uuid.UUID
	Role    string
}

type Command struct {
	MessageID     uuid.UUID
	EventType     string
	EvidenceID    uuid.UUID
	LotID         uuid.UUID
	Stage         string
	QuarantineKey string
	SizeBytes     int64
	Uploader      Uploader
}

type UseCase interface {
	usecases.UseCaseWithoutReturn[Command]
}
