package projectlotstage

import (
	"time"

	"github.com/google/uuid"

	"rutadeorigen/qr/internal/application/usecases"
	"rutadeorigen/qr/internal/domain/lotview"
)

type Command struct {
	MessageID  uuid.UUID
	EventType  string
	LotID      uuid.UUID
	Stage      lotview.Stage
	Version    int
	OccurredAt time.Time
}

type UseCase interface {
	usecases.UseCaseWithoutReturn[Command]
}
