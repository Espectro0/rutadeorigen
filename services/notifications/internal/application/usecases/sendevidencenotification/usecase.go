package sendevidencenotification

import (
	"github.com/google/uuid"

	"rutadeorigen/notifications/internal/application/usecases"
	"rutadeorigen/notifications/internal/domain/notification"
)

type Command struct {
	MessageID      uuid.UUID
	EventType      string
	Kind           notification.Kind
	EvidenceID     uuid.UUID
	LotID          uuid.UUID
	LotCode        string
	Stage          string
	FileName       string
	Reason         string
	RecipientName  string
	RecipientEmail string
}

type UseCase interface {
	usecases.UseCaseWithoutReturn[Command]
}
