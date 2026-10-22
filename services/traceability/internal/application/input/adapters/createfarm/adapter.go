package createfarm

import (
	"context"

	"rutadeorigen/traceability/internal/application/input/assembler"
	"rutadeorigen/traceability/internal/application/input/dto/request"
	"rutadeorigen/traceability/internal/application/input/dto/response"
	"rutadeorigen/traceability/internal/application/input/ports"
	createfarmuc "rutadeorigen/traceability/internal/application/usecases/createfarm"
	"rutadeorigen/traceability/internal/domain/farm"
)

var _ ports.CreateFarmPort = (*Adapter)(nil)

type Adapter struct {
	useCase   createfarmuc.UseCase
	assembler assembler.DTOAssembler[*farm.Farm, request.CreateFarm, response.Farm]
}

func New(useCase createfarmuc.UseCase) *Adapter {
	return &Adapter{useCase: useCase, assembler: Assembler{}}
}

func (a *Adapter) Execute(ctx context.Context, input request.CreateFarm) (response.Farm, error) {
	draft, err := a.assembler.ToDomain(input)
	if err != nil {
		return response.Farm{}, err
	}

	if err := a.useCase.Execute(ctx, draft); err != nil {
		return response.Farm{}, err
	}

	return a.assembler.ToDTO(draft), nil
}
