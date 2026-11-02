package listpublicevidences

import (
	"context"

	"rutadeorigen/qr/internal/application/output/ports"
	"rutadeorigen/qr/internal/domain/evidenceview"
	"rutadeorigen/qr/internal/domain/lotview"
	"rutadeorigen/qr/internal/platform/exception"
)

var _ UseCase = (*ListPublicEvidences)(nil)

type ListPublicEvidences struct {
	evidences ports.EvidenceViewRepository
	viewer    ports.EvidenceViewer
	settings  Settings
}

func New(evidences ports.EvidenceViewRepository, viewer ports.EvidenceViewer, settings Settings) *ListPublicEvidences {
	return &ListPublicEvidences{evidences: evidences, viewer: viewer, settings: settings}
}

func (uc *ListPublicEvidences) Execute(ctx context.Context, view *lotview.LotView) ([]Item, error) {
	if view.IsDefault() || !view.IsActive() {
		return []Item{}, nil
	}

	stored, err := uc.evidences.ListByLotID(ctx, view.LotID(), uc.settings.MaxPerLot)
	if err != nil {
		return []Item{}, exception.Wrap(err, "list evidence views")
	}

	items := make([]Item, 0, len(stored))
	for _, evidence := range stored {
		item, signed := uc.signedItem(ctx, evidence)
		if signed {
			items = append(items, item)
		}
	}
	return items, nil
}

func (uc *ListPublicEvidences) signedItem(ctx context.Context, evidence *evidenceview.EvidenceView) (Item, bool) {
	thumbnail, err := uc.viewer.PresignView(ctx, evidence.ThumbnailKey(), uc.settings.ViewTTL)
	if err != nil {
		return Item{}, false
	}
	image, err := uc.viewer.PresignView(ctx, evidence.FinalKey(), uc.settings.ViewTTL)
	if err != nil {
		return Item{}, false
	}

	return Item{
		Evidence:     evidence,
		ThumbnailURL: thumbnail.URL,
		ImageURL:     image.URL,
		URLExpiresAt: thumbnail.ExpiresAt,
	}, true
}
