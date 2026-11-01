package listlotevents

import (
	"context"

	"github.com/google/uuid"

	"rutadeorigen/lib/requestcontext"
	"rutadeorigen/traceability/internal/application/output/ports"
	"rutadeorigen/traceability/internal/application/usecases/validators"
	"rutadeorigen/traceability/internal/domain/lot"
	"rutadeorigen/traceability/internal/platform/exception"
)

var _ UseCase = (*ListLotEvents)(nil)

type ListLotEvents struct {
	actors ports.ActorRepository
	lots   ports.LotRepository
	events ports.LotEventRepository
	auth   ports.Authorizer
}

func New(actors ports.ActorRepository, lots ports.LotRepository, events ports.LotEventRepository, auth ports.Authorizer) *ListLotEvents {
	return &ListLotEvents{actors: actors, lots: lots, events: events, auth: auth}
}

func (uc *ListLotEvents) Execute(ctx context.Context, id uuid.UUID) (Result, error) {
	a, err := validators.ActorActive(ctx, uc.actors, requestcontext.ActorID(ctx))
	if err != nil {
		return Result{}, err
	}

	l, err := validators.LotExists(ctx, uc.lots, id)
	if err != nil {
		return Result{}, err
	}

	allowed, err := uc.auth.IsAllowed(ctx,
		ports.Subject{ActorID: a.ID(), Role: a.Type()},
		ports.ActionReadLot,
		ports.Resource{Type: ports.ResourceLot, ID: l.ID(), CustodianID: l.CustodianID()})
	if err != nil {
		return Result{}, exception.Wrap(err, "evaluate read lot policy")
	}
	if !allowed {
		return Result{}, exception.New(exception.CodeForbidden)
	}

	events, err := uc.events.ListByLotID(ctx, l.ID())
	if err != nil {
		return Result{}, exception.Wrap(err, "list lot events")
	}
	if events == nil {
		events = []*lot.Event{}
	}
	return Result{Events: events, ChainValid: lot.VerifyChain(l.ID(), events)}, nil
}
