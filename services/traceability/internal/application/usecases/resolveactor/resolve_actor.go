package resolveactor

import (
	"context"

	"rutadeorigen/traceability/internal/application/output/ports"
	"rutadeorigen/traceability/internal/domain/actor"
	"rutadeorigen/traceability/internal/platform/exception"
	"rutadeorigen/traceability/internal/platform/helper"
)

var _ UseCase = (*ResolveActor)(nil)

type ResolveActor struct {
	actors ports.ActorRepository
}

func New(actors ports.ActorRepository) *ResolveActor {
	return &ResolveActor{actors: actors}
}

func (uc *ResolveActor) Execute(ctx context.Context, userID string) (*actor.Actor, error) {
	userID = helper.ApplyTrim(userID)
	if helper.IsEmpty(userID) {
		return actor.Default(), exception.New(exception.CodeUnauthorized)
	}

	a, err := uc.actors.FindByUserID(ctx, userID)
	if err != nil {
		return actor.Default(), exception.Wrap(err, "find actor by user id")
	}
	if a.IsDefault() {
		return actor.Default(), exception.New(exception.CodeActorNotFound)
	}
	if !a.IsActive() {
		return actor.Default(), exception.New(exception.CodeForbidden)
	}

	return a, nil
}
