package registeractor

import (
	"context"

	"rutadeorigen/lib/requestcontext"
	"rutadeorigen/traceability/internal/application/input/adapters/actormapper"
	"rutadeorigen/traceability/internal/application/input/dto/request"
	"rutadeorigen/traceability/internal/application/input/dto/response"
	"rutadeorigen/traceability/internal/application/input/ports"
	registeractoruc "rutadeorigen/traceability/internal/application/usecases/registeractor"
	"rutadeorigen/traceability/internal/domain/actor"
)

var _ ports.RegisterActorPort = (*Adapter)(nil)

type Adapter struct{ useCase registeractoruc.UseCase }

func New(useCase registeractoruc.UseCase) *Adapter { return &Adapter{useCase: useCase} }

func (a *Adapter) Execute(ctx context.Context, input request.RegisterActor) (response.ActorRegistration, error) {
	res, err := a.useCase.Execute(ctx, registeractoruc.Command{
		UserID: requestcontext.UserID(ctx),
		Email:  requestcontext.Email(ctx),
		Type:   actor.Type(input.Type),
		Name:   input.Name,
	})
	if err != nil {
		return response.ActorRegistration{}, err
	}
	return response.ActorRegistration{Actor: actormapper.ToResponse(res.Actor), Created: res.Created}, nil
}
