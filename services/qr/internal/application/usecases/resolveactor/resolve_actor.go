package resolveactor

import (
	"context"

	"rutadeorigen/qr/internal/application/output/ports"
	"rutadeorigen/qr/internal/domain/actorview"
	"rutadeorigen/qr/internal/platform/exception"
	"rutadeorigen/qr/internal/platform/helper"
)

var _ UseCase = (*ResolveActor)(nil)

type ResolveActor struct {
	actors ports.ActorViewRepository
}

func New(actors ports.ActorViewRepository) *ResolveActor {
	return &ResolveActor{actors: actors}
}

func (uc *ResolveActor) Execute(ctx context.Context, cmd Command) (*actorview.ActorView, error) {
	userID := helper.ApplyTrim(cmd.UserID)

	var (
		view *actorview.ActorView
		err  error
	)
	switch {
	case !helper.IsEmpty(userID):
		view, err = uc.actors.FindByUserID(ctx, userID)
	case !helper.IsDefaultUUID(cmd.ActorID):
		view, err = uc.actors.FindByID(ctx, cmd.ActorID)
	default:
		return actorview.Default(), exception.New(exception.CodeUnauthorized)
	}
	if err != nil {
		return actorview.Default(), exception.Wrap(err, "find actor view")
	}
	if view.IsDefault() {
		return actorview.Default(), exception.New(exception.CodeActorViewNotFound)
	}
	return view, nil
}
