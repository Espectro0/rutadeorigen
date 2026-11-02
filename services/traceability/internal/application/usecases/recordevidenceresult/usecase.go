package recordevidenceresult

import (
	"time"

	"github.com/google/uuid"

	"rutadeorigen/traceability/internal/application/usecases"
)

type Outcome string

const (
	OutcomeProcessed Outcome = "processed"
	OutcomeRejected  Outcome = "rejected"
)

type Command struct {
	EvidenceID   uuid.UUID
	Outcome      Outcome
	SHA256       string
	FinalKey     string
	ThumbnailKey string
	Reason       string
	At           time.Time
}

type UseCase interface {
	usecases.UseCaseWithoutReturn[Command]
}
