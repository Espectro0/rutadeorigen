package listactors

import (
	"context"

	"rutadeorigen/lib/requestcontext"
	"rutadeorigen/traceability/internal/application/output/ports"
	"rutadeorigen/traceability/internal/application/usecases/validators"
	"rutadeorigen/traceability/internal/domain/actor"
	"rutadeorigen/traceability/internal/platform/exception"
)

var _ UseCase = (*ListActors)(nil)

type ListActors struct {
	actors ports.ActorRepository
	auth   ports.Authorizer
}

func New(actors ports.ActorRepository, auth ports.Authorizer) *ListActors {
	return &ListActors{actors: actors, auth: auth}
}

func (uc *ListActors) Execute(ctx context.Context, q Query) ([]*actor.Actor, error) {
	a, err := validators.ActorActive(ctx, uc.actors, requestcontext.ActorID(ctx))
	if err != nil {
		return []*actor.Actor{}, err
	}

	allowed, err := uc.auth.IsAllowed(ctx,
		ports.Subject{ActorID: a.ID(), Role: a.Type()},
		ports.ActionReadActor,
		ports.Resource{Type: ports.ResourceActor})
	if err != nil {
		return []*actor.Actor{}, exception.Wrap(err, "evaluate list actors policy")
	}
	if !allowed {
		return []*actor.Actor{}, exception.New(exception.CodeForbidden)
	}

	if !q.Type.IsValid() {
		return []*actor.Actor{}, exception.New(exception.CodeActorInvalidType)
	}

	found, err := uc.actors.ListByType(ctx, q.Type)
	if err != nil {
		return []*actor.Actor{}, exception.Wrap(err, "list actors by type")
	}

	active := make([]*actor.Actor, 0, len(found))
	for _, candidate := range found {
		if !candidate.IsDefault() && candidate.IsActive() {
			active = append(active, candidate)
		}
	}
	return active, nil
}
