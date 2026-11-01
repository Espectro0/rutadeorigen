package listactors

import (
	"context"

	"rutadeorigen/traceability/internal/application/input/adapters/actormapper"
	"rutadeorigen/traceability/internal/application/input/dto/request"
	"rutadeorigen/traceability/internal/application/input/dto/response"
	"rutadeorigen/traceability/internal/application/input/ports"
	listactorsuc "rutadeorigen/traceability/internal/application/usecases/listactors"
	"rutadeorigen/traceability/internal/domain/actor"
)

var _ ports.ListActorsPort = (*Adapter)(nil)

type Adapter struct {
	useCase listactorsuc.UseCase
}

func New(useCase listactorsuc.UseCase) *Adapter {
	return &Adapter{useCase: useCase}
}

func (a *Adapter) Execute(ctx context.Context, input request.ListActors) (response.ActorList, error) {
	actors, err := a.useCase.Execute(ctx, listactorsuc.Query{Type: actor.ParseType(input.Type)})
	if err != nil {
		return response.ActorList{Items: []response.ActorSummary{}}, err
	}
	return actormapper.ToList(actors), nil
}
