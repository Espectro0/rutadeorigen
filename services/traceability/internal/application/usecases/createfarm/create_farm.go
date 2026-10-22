package createfarm

import (
	"context"

	"rutadeorigen/lib/requestcontext"
	"rutadeorigen/traceability/internal/application/output/ports"
	"rutadeorigen/traceability/internal/application/usecases/validators"
	"rutadeorigen/traceability/internal/domain/actor"
	"rutadeorigen/traceability/internal/domain/farm"
	"rutadeorigen/traceability/internal/platform/exception"
	"rutadeorigen/traceability/internal/platform/helper"
)

var _ UseCase = (*CreateFarm)(nil)

type CreateFarm struct {
	actors ports.ActorRepository
	farms  ports.FarmRepository
	auth   ports.Authorizer
}

func New(actors ports.ActorRepository, farms ports.FarmRepository, auth ports.Authorizer) *CreateFarm {
	return &CreateFarm{actors: actors, farms: farms, auth: auth}
}

func (uc *CreateFarm) Execute(ctx context.Context, draft *farm.Farm) error {
	if helper.IsNil(draft) {
		draft = farm.Default()
	}

	a, err := validators.ActorActive(ctx, uc.actors, requestcontext.ActorID(ctx))
	if err != nil {
		return err
	}

	if !a.IsProducer() {
		return exception.New(exception.CodeFarmInvalidProducer)
	}

	if err := uc.authorize(ctx, a); err != nil {
		return err
	}

	created, err := farm.New(a.ID(), draft.Name(), draft.Altitude())
	if err != nil {
		return err
	}

	if err := uc.farms.Save(ctx, created); err != nil {
		return exception.Wrap(err, "save farm")
	}

	*draft = *created
	return nil
}

func (uc *CreateFarm) authorize(ctx context.Context, a *actor.Actor) error {
	allowed, err := uc.auth.IsAllowed(ctx,
		ports.Subject{ActorID: a.ID(), Role: a.Type()},
		ports.ActionCreateFarm,
		ports.Resource{Type: ports.ResourceFarm, OwnerID: a.ID()})
	if err != nil {
		return exception.Wrap(err, "evaluate create farm policy")
	}
	if !allowed {
		return exception.New(exception.CodeForbidden)
	}
	return nil
}
