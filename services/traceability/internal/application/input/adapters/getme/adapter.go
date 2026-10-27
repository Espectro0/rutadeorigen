package getme

import (
	"context"

	"rutadeorigen/lib/requestcontext"
	"rutadeorigen/traceability/internal/application/input/adapters/actormapper"
	"rutadeorigen/traceability/internal/application/input/dto/request"
	"rutadeorigen/traceability/internal/application/input/dto/response"
	"rutadeorigen/traceability/internal/application/input/ports"
	resolveactoruc "rutadeorigen/traceability/internal/application/usecases/resolveactor"
)

var _ ports.GetMePort = (*Adapter)(nil)

type Adapter struct{ useCase resolveactoruc.UseCase }

func New(useCase resolveactoruc.UseCase) *Adapter { return &Adapter{useCase: useCase} }

func (a *Adapter) Execute(ctx context.Context, _ request.GetMe) (response.Actor, error) {
	found, err := a.useCase.Execute(ctx, requestcontext.UserID(ctx))
	if err != nil {
		return response.Actor{}, err
	}
	return actormapper.ToResponse(found), nil
}
