package listpublicevidences

import (
	"time"

	"rutadeorigen/qr/internal/application/usecases"
	"rutadeorigen/qr/internal/domain/evidenceview"
	"rutadeorigen/qr/internal/domain/lotview"
)

type Settings struct {
	ViewTTL   time.Duration
	MaxPerLot int
}

type Item struct {
	Evidence     *evidenceview.EvidenceView
	ThumbnailURL string
	ImageURL     string
	URLExpiresAt time.Time
}

type UseCase interface {
	usecases.UseCaseWithReturn[*lotview.LotView, []Item]
}
