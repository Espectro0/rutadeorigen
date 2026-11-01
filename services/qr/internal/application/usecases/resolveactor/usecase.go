package resolveactor

import (
	"github.com/google/uuid"

	"rutadeorigen/qr/internal/application/usecases"
	"rutadeorigen/qr/internal/domain/actorview"
)

// Command identifies the actor by the user of the session or, when there is
// no user, by the actor id itself (development header mode).
type Command struct {
	UserID  string
	ActorID uuid.UUID
}

type UseCase interface {
	usecases.UseCaseWithReturn[Command, *actorview.ActorView]
}
