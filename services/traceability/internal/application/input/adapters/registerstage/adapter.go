package registerstage

import (
	"context"

	"rutadeorigen/traceability/internal/application/input/dto/request"
	"rutadeorigen/traceability/internal/application/input/dto/response"
	"rutadeorigen/traceability/internal/application/input/ports"
	registerstageuc "rutadeorigen/traceability/internal/application/usecases/registerstage"
)

var _ ports.RegisterStagePort = (*Adapter)(nil)

type Adapter struct {
	useCase   registerstageuc.UseCase
	assembler Assembler
}

func New(useCase registerstageuc.UseCase) *Adapter {
	return &Adapter{useCase: useCase, assembler: Assembler{}}
}

func (a *Adapter) Execute(ctx context.Context, input request.RegisterStage) (response.RegisteredStage, error) {
	cmd, err := a.assembler.ToCommand(input)
	if err != nil {
		return response.RegisteredStage{}, err
	}

	res, err := a.useCase.Execute(ctx, cmd)
	if err != nil {
		return response.RegisteredStage{}, err
	}

	return a.assembler.ToDTO(res), nil
}
