package listfarm

import (
	"context"

	"rutadeorigen/lib/requestcontext"
	"rutadeorigen/traceability/internal/application/output/ports"
	"rutadeorigen/traceability/internal/application/usecases/validators"
	"rutadeorigen/traceability/internal/domain/farm"
	"rutadeorigen/traceability/internal/platform/exception"
)

var _ UseCase = (*ListFarms)(nil)

type ListFarms struct {
	actors ports.ActorRepository
	farms  ports.FarmRepository
	auth   ports.Authorizer
}

func New(actors ports.ActorRepository, farms ports.FarmRepository, auth ports.Authorizer) *ListFarms {
	return &ListFarms{actors: actors, farms: farms, auth: auth}
}

func (uc *ListFarms) Execute(ctx context.Context, _ Query) ([]*farm.Farm, error) {
	a, err := validators.ActorActive(ctx, uc.actors, requestcontext.ActorID(ctx))
	if err != nil {
		return []*farm.Farm{}, err
	}

	allowed, err := uc.auth.IsAllowed(ctx,
		ports.Subject{ActorID: a.ID(), Role: a.Type()},
		ports.ActionReadFarm,
		ports.Resource{Type: ports.ResourceFarm, OwnerID: a.ID()})
	if err != nil {
		return []*farm.Farm{}, exception.Wrap(err, "evaluate list farms policy")
	}
	if !allowed {
		return []*farm.Farm{}, exception.New(exception.CodeForbidden)
	}

	farms, err := uc.farms.ListByProducer(ctx, a.ID())
	if err != nil {
		return []*farm.Farm{}, exception.Wrap(err, "list farms by producer")
	}
	if farms == nil {
		return []*farm.Farm{}, nil
	}
	return farms, nil
}
