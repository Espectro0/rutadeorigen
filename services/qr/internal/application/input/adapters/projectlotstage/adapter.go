package projectlotstage

import (
	"context"

	"rutadeorigen/qr/internal/application/input/assembler"
	"rutadeorigen/qr/internal/application/input/dto/request"
	"rutadeorigen/qr/internal/application/input/dto/response"
	"rutadeorigen/qr/internal/application/input/ports"
	projectlotstageuc "rutadeorigen/qr/internal/application/usecases/projectlotstage"
)

var _ ports.ProjectLotStagePort = (*Adapter)(nil)

type Adapter struct {
	useCase   projectlotstageuc.UseCase
	assembler assembler.DTOAssembler[projectlotstageuc.Command, request.ProjectLotStage, response.Empty]
}

func New(useCase projectlotstageuc.UseCase) *Adapter {
	return &Adapter{
		useCase:   useCase,
		assembler: Assembler{},
	}
}

func (a *Adapter) Execute(ctx context.Context, input request.ProjectLotStage) (response.Empty, error) {
	cmd, err := a.assembler.ToDomain(input)
	if err != nil {
		return response.Empty{}, err
	}

	if err := a.useCase.Execute(ctx, cmd); err != nil {
		return response.Empty{}, err
	}

	return a.assembler.ToDTO(cmd), nil
}
