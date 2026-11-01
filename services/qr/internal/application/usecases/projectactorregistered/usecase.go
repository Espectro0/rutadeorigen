package projectactorregistered

import (
	"time"

	"github.com/google/uuid"

	"rutadeorigen/qr/internal/application/usecases"
	"rutadeorigen/qr/internal/domain/actorview"
)

type Command struct {
	MessageID  uuid.UUID
	EventType  string
	ActorID    uuid.UUID
	UserID     string
	Role       actorview.Role
	OccurredAt time.Time
}

type UseCase interface {
	usecases.UseCaseWithoutReturn[Command]
}
