package getlot

import (
	"context"

	"github.com/google/uuid"

	"rutadeorigen/lib/requestcontext"
	"rutadeorigen/traceability/internal/application/output/ports"
	"rutadeorigen/traceability/internal/application/usecases/validators"
	"rutadeorigen/traceability/internal/domain/lot"
	"rutadeorigen/traceability/internal/platform/exception"
)

var _ UseCase = (*GetLot)(nil)

type GetLot struct {
	actors ports.ActorRepository
	lots   ports.LotRepository
	auth   ports.Authorizer
}

func New(actors ports.ActorRepository, lots ports.LotRepository, auth ports.Authorizer) *GetLot {
	return &GetLot{actors: actors, lots: lots, auth: auth}
}

func (uc *GetLot) Execute(ctx context.Context, id uuid.UUID) (*lot.Lot, error) {
	a, err := validators.ActorActive(ctx, uc.actors, requestcontext.ActorID(ctx))
	if err != nil {
		return lot.Default(), err
	}

	l, err := validators.LotExists(ctx, uc.lots, id)
	if err != nil {
		return lot.Default(), err
	}

	allowed, err := uc.auth.IsAllowed(ctx,
		ports.Subject{ActorID: a.ID(), Role: a.Type()},
		ports.ActionReadLot,
		ports.Resource{Type: ports.ResourceLot, ID: l.ID(), CustodianID: l.CustodianID()})
	if err != nil {
		return lot.Default(), exception.Wrap(err, "evaluate read lot policy")
	}
	if !allowed {
		return lot.Default(), exception.New(exception.CodeForbidden)
	}
	return l, nil
}
