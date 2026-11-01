package projectlotvoided

import (
	"context"

	"rutadeorigen/qr/internal/application/input/assembler"
	"rutadeorigen/qr/internal/application/input/dto/request"
	"rutadeorigen/qr/internal/application/input/dto/response"
	"rutadeorigen/qr/internal/application/input/ports"
	projectlotvoideduc "rutadeorigen/qr/internal/application/usecases/projectlotvoided"
)

var _ ports.ProjectLotVoidedPort = (*Adapter)(nil)

type Adapter struct {
	useCase   projectlotvoideduc.UseCase
	assembler assembler.DTOAssembler[projectlotvoideduc.Command, request.ProjectLotVoided, response.Empty]
}

func New(useCase projectlotvoideduc.UseCase) *Adapter {
	return &Adapter{
		useCase:   useCase,
		assembler: Assembler{},
	}
}

func (a *Adapter) Execute(ctx context.Context, input request.ProjectLotVoided) (response.Empty, error) {
	cmd, err := a.assembler.ToDomain(input)
	if err != nil {
		return response.Empty{}, err
	}

	if err := a.useCase.Execute(ctx, cmd); err != nil {
		return response.Empty{}, err
	}

	return a.assembler.ToDTO(cmd), nil
}
