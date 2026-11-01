package projectlottransferred

import (
	"time"

	"github.com/google/uuid"

	"rutadeorigen/qr/internal/application/usecases"
)

type Command struct {
	MessageID   uuid.UUID
	EventType   string
	LotID       uuid.UUID
	CustodianID uuid.UUID
	Version     int
	OccurredAt  time.Time
}

type UseCase interface {
	usecases.UseCaseWithoutReturn[Command]
}
