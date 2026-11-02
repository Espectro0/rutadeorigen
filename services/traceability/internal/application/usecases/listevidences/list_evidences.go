package listevidences

import (
	"context"

	"github.com/google/uuid"

	"rutadeorigen/lib/requestcontext"
	"rutadeorigen/traceability/internal/application/output/ports"
	"rutadeorigen/traceability/internal/application/usecases/validators"
	"rutadeorigen/traceability/internal/domain/evidence"
	"rutadeorigen/traceability/internal/platform/exception"
	"rutadeorigen/traceability/internal/platform/helper"
)

var _ UseCase = (*ListEvidences)(nil)

type ListEvidences struct {
	actors    ports.ActorRepository
	lots      ports.LotRepository
	evidences ports.EvidenceRepository
	auth      ports.Authorizer
	viewer    ports.EvidenceViewer
	settings  Settings
}

func New(actors ports.ActorRepository, lots ports.LotRepository, evidences ports.EvidenceRepository,
	auth ports.Authorizer, viewer ports.EvidenceViewer, settings Settings) *ListEvidences {
	return &ListEvidences{actors: actors, lots: lots, evidences: evidences, auth: auth, viewer: viewer, settings: settings}
}

func (uc *ListEvidences) Execute(ctx context.Context, lotID uuid.UUID) ([]Item, error) {
	a, err := validators.ActorActive(ctx, uc.actors, requestcontext.ActorID(ctx))
	if err != nil {
		return []Item{}, err
	}

	l, err := validators.LotExists(ctx, uc.lots, lotID)
	if err != nil {
		return []Item{}, err
	}

	allowed, err := uc.auth.IsAllowed(ctx,
		ports.Subject{ActorID: a.ID(), Role: a.Type()},
		ports.ActionReadLot,
		ports.Resource{Type: ports.ResourceLot, ID: l.ID(), CustodianID: l.CustodianID()})
	if err != nil {
		return []Item{}, exception.Wrap(err, "evaluate read lot policy")
	}
	if !allowed {
		return []Item{}, exception.New(exception.CodeForbidden)
	}

	items, err := uc.evidences.ListByLotID(ctx, l.ID())
	if err != nil {
		return []Item{}, exception.Wrap(err, "list evidences")
	}

	result := make([]Item, 0, len(items))
	for _, e := range items {
		result = append(result, uc.itemWithViewURLsBestEffort(ctx, e))
	}
	return result, nil
}

func (uc *ListEvidences) itemWithViewURLsBestEffort(ctx context.Context, e *evidence.Evidence) Item {
	item := Item{Evidence: e, URLExpiresAt: helper.DefaultDate()}
	if e.Status() != evidence.StatusProcessed || helper.IsEmpty(e.ThumbnailKey()) || helper.IsEmpty(e.FinalKey()) {
		return item
	}

	thumbnail, err := uc.viewer.PresignView(ctx, e.ThumbnailKey(), uc.settings.ViewTTL)
	if err != nil {
		return item
	}
	image, err := uc.viewer.PresignView(ctx, e.FinalKey(), uc.settings.ViewTTL)
	if err != nil {
		return item
	}

	item.ThumbnailURL = thumbnail.URL
	item.ImageURL = image.URL
	item.URLExpiresAt = thumbnail.ExpiresAt
	return item
}
