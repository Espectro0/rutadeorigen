package resolveactor

import (
	"context"

	"rutadeorigen/qr/internal/application/input/dto/request"
	"rutadeorigen/qr/internal/application/input/dto/response"
	"rutadeorigen/qr/internal/application/input/ports"
	resolveactoruc "rutadeorigen/qr/internal/application/usecases/resolveactor"
	"rutadeorigen/qr/internal/platform/helper"
)

var _ ports.ResolveActorPort = (*Adapter)(nil)

type Adapter struct {
	useCase resolveactoruc.UseCase
}

func New(useCase resolveactoruc.UseCase) *Adapter {
	return &Adapter{useCase: useCase}
}

func (a *Adapter) Execute(ctx context.Context, input request.ResolveActor) (response.ResolvedActor, error) {
	view, err := a.useCase.Execute(ctx, resolveactoruc.Command{
		UserID:  input.UserID,
		ActorID: helper.ParseUUID(input.ActorID),
	})
	if err != nil {
		return response.ResolvedActor{}, err
	}

	return response.ResolvedActor{ActorID: view.ActorID(), Role: view.Role().String()}, nil
}
