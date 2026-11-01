package getlot

import (
	"context"

	"github.com/google/uuid"

	"rutadeorigen/lib/requestcontext"
	"rutadeorigen/traceability/internal/application/output/ports"
	"rutadeorigen/traceability/internal/application/usecases/validators"
	"rutadeorigen/traceability/internal/domain/actor"
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

func (uc *GetLot) Execute(ctx context.Context, id uuid.UUID) (Result, error) {
	failed := Result{Lot: lot.Default()}

	a, err := validators.ActorActive(ctx, uc.actors, requestcontext.ActorID(ctx))
	if err != nil {
		return failed, err
	}

	l, err := validators.LotExists(ctx, uc.lots, id)
	if err != nil {
		return failed, err
	}

	allowed, err := uc.isAllowed(ctx, a, ports.ActionReadLot, resourceOf(l, lot.StageNone))
	if err != nil {
		return failed, err
	}
	if !allowed {
		return failed, exception.New(exception.CodeForbidden)
	}

	actions, err := uc.allowedActions(ctx, a, l)
	if err != nil {
		return failed, err
	}
	return Result{Lot: l, AllowedActions: actions}, nil
}

func (uc *GetLot) allowedActions(ctx context.Context, a *actor.Actor, l *lot.Lot) ([]ports.Action, error) {
	actions := []ports.Action{}
	for _, c := range candidateActions(l) {
		allowed, err := uc.isAllowed(ctx, a, c.action, resourceOf(l, c.stage))
		if err != nil {
			return nil, err
		}
		if allowed {
			actions = append(actions, c.action)
		}
	}
	return actions, nil
}

func (uc *GetLot) isAllowed(ctx context.Context, a *actor.Actor, action ports.Action, resource ports.Resource) (bool, error) {
	allowed, err := uc.auth.IsAllowed(ctx, ports.Subject{ActorID: a.ID(), Role: a.Type()}, action, resource)
	if err != nil {
		return false, exception.Wrap(err, "evaluate "+string(action)+" policy")
	}
	return allowed, nil
}

func resourceOf(l *lot.Lot, stage lot.Stage) ports.Resource {
	return ports.Resource{Type: ports.ResourceLot, ID: l.ID(), CustodianID: l.CustodianID(), Stage: stage}
}
