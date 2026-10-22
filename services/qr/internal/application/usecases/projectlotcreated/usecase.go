package projectlotcreated

import (
	"github.com/google/uuid"

	"rutadeorigen/qr/internal/application/usecases"
	"rutadeorigen/qr/internal/domain/lotview"
)

type Command struct {
	MessageID uuid.UUID
	EventType string
	Lot       lotview.NewParams
}

type UseCase interface {
	usecases.UseCaseWithoutReturn[Command]
}
