package listfarm

import (
	"context"

	"rutadeorigen/traceability/internal/application/input/adapters/farmmapper"
	"rutadeorigen/traceability/internal/application/input/dto/request"
	"rutadeorigen/traceability/internal/application/input/dto/response"
	"rutadeorigen/traceability/internal/application/input/ports"
	listfarmuc "rutadeorigen/traceability/internal/application/usecases/listfarm"
)

var _ ports.ListFarmsPort = (*Adapter)(nil)

type Adapter struct {
	useCase listfarmuc.UseCase
}

func New(useCase listfarmuc.UseCase) *Adapter {
	return &Adapter{useCase: useCase}
}

func (a *Adapter) Execute(ctx context.Context, _ request.ListFarms) (response.FarmList, error) {
	farms, err := a.useCase.Execute(ctx, listfarmuc.Query{})
	if err != nil {
		return response.FarmList{Items: []response.Farm{}}, err
	}
	return farmmapper.ToList(farms), nil
}
