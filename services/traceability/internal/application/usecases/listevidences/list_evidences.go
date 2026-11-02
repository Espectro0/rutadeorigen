package listevidences

import (
	"context"

	"github.com/google/uuid"

	"rutadeorigen/lib/requestcontext"
	"rutadeorigen/traceability/internal/application/output/ports"
	"rutadeorigen/traceability/internal/application/usecases/validators"
	"rutadeorigen/traceability/internal/domain/evidence"
	"rutadeorigen/traceability/internal/platform/exception"
)

var _ UseCase = (*ListEvidences)(nil)

type ListEvidences struct {
	actors    ports.ActorRepository
	lots      ports.LotRepository
	evidences ports.EvidenceRepository
	auth      ports.Authorizer
}

func New(actors ports.ActorRepository, lots ports.LotRepository, evidences ports.EvidenceRepository, auth ports.Authorizer) *ListEvidences {
	return &ListEvidences{actors: actors, lots: lots, evidences: evidences, auth: auth}
}

func (uc *ListEvidences) Execute(ctx context.Context, lotID uuid.UUID) ([]*evidence.Evidence, error) {
	a, err := validators.ActorActive(ctx, uc.actors, requestcontext.ActorID(ctx))
	if err != nil {
		return []*evidence.Evidence{}, err
	}

	l, err := validators.LotExists(ctx, uc.lots, lotID)
	if err != nil {
		return []*evidence.Evidence{}, err
	}

	allowed, err := uc.auth.IsAllowed(ctx,
		ports.Subject{ActorID: a.ID(), Role: a.Type()},
		ports.ActionReadLot,
		ports.Resource{Type: ports.ResourceLot, ID: l.ID(), CustodianID: l.CustodianID()})
	if err != nil {
		return []*evidence.Evidence{}, exception.Wrap(err, "evaluate read lot policy")
	}
	if !allowed {
		return []*evidence.Evidence{}, exception.New(exception.CodeForbidden)
	}

	items, err := uc.evidences.ListByLotID(ctx, l.ID())
	if err != nil {
		return []*evidence.Evidence{}, exception.Wrap(err, "list evidences")
	}
	if items == nil {
		items = []*evidence.Evidence{}
	}
	return items, nil
}
