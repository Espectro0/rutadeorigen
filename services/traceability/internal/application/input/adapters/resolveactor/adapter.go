package resolveactor

import (
	"context"

	"rutadeorigen/lib/requestcontext"
	"rutadeorigen/traceability/internal/application/input/dto/request"
	"rutadeorigen/traceability/internal/application/input/dto/response"
	"rutadeorigen/traceability/internal/application/input/ports"
	resolveactoruc "rutadeorigen/traceability/internal/application/usecases/resolveactor"
)

var _ ports.ResolveActorPort = (*Adapter)(nil)

type Adapter struct{ useCase resolveactoruc.UseCase }

func New(useCase resolveactoruc.UseCase) *Adapter { return &Adapter{useCase: useCase} }

func (a *Adapter) Execute(ctx context.Context, _ request.ResolveActor) (response.ResolvedActor, error) {
	found, err := a.useCase.Execute(ctx, requestcontext.UserID(ctx))
	if err != nil {
		return response.ResolvedActor{}, err
	}
	return response.ResolvedActor{ID: found.ID()}, nil
}
