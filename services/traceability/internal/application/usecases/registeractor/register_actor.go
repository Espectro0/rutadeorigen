package registeractor

import (
	"context"

	"rutadeorigen/traceability/internal/application/output/ports"
	"rutadeorigen/traceability/internal/domain/actor"
	"rutadeorigen/traceability/internal/platform/exception"
	"rutadeorigen/traceability/internal/platform/helper"
)

var _ UseCase = (*RegisterActor)(nil)

type RegisterActor struct {
	actors ports.ActorRepository
}

func New(actors ports.ActorRepository) *RegisterActor {
	return &RegisterActor{actors: actors}
}

func (uc *RegisterActor) Execute(ctx context.Context, cmd Command) (Result, error) {
	userID := helper.ApplyTrim(cmd.UserID)
	if helper.IsEmpty(userID) {
		return Result{}, exception.New(exception.CodeUnauthorized)
	}

	existing, err := uc.actors.FindByUserID(ctx, userID)
	if err != nil {
		return Result{}, exception.Wrap(err, "find actor by user id")
	}
	if !existing.IsDefault() {
		return Result{Actor: existing}, nil
	}

	created, err := actor.New(userID, cmd.Type, cmd.Name, cmd.Email)
	if err != nil {
		return Result{}, err
	}

	if err := uc.actors.Save(ctx, created); err != nil {
		if exception.HasCode(err, exception.CodeActorAlreadyExists) {
			return uc.findAfterConflict(ctx, userID)
		}
		return Result{}, exception.Wrap(err, "save actor")
	}

	return Result{Actor: created, Created: true}, nil
}

func (uc *RegisterActor) findAfterConflict(ctx context.Context, userID string) (Result, error) {
	found, err := uc.actors.FindByUserID(ctx, userID)
	if err != nil {
		return Result{}, exception.Wrap(err, "find actor after conflict")
	}
	if found.IsDefault() {
		return Result{}, exception.New(exception.CodeInternal).WithTechnical("actor conflict without an existing actor")
	}
	return Result{Actor: found}, nil
}
