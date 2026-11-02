package projectevidenceprocessed

import (
	"time"

	"github.com/google/uuid"

	"rutadeorigen/qr/internal/application/usecases"
)

type Command struct {
	MessageID    uuid.UUID
	EventType    string
	EvidenceID   uuid.UUID
	LotID        uuid.UUID
	Stage        string
	FinalKey     string
	ThumbnailKey string
	ContentType  string
	ProcessedAt  time.Time
}

type UseCase interface {
	usecases.UseCaseWithoutReturn[Command]
}
