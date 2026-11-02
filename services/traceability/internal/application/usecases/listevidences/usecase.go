package listevidences

import (
	"time"

	"github.com/google/uuid"

	"rutadeorigen/traceability/internal/application/usecases"
	"rutadeorigen/traceability/internal/domain/evidence"
)

type Settings struct {
	ViewTTL time.Duration
}

type Item struct {
	Evidence     *evidence.Evidence
	ThumbnailURL string
	ImageURL     string
	URLExpiresAt time.Time
}

type UseCase interface {
	usecases.UseCaseWithReturn[uuid.UUID, []Item]
}
