package projectlotcreated

import (
	"context"

	"rutadeorigen/qr/internal/application/input/assembler"
	"rutadeorigen/qr/internal/application/input/dto/request"
	"rutadeorigen/qr/internal/application/input/dto/response"
	"rutadeorigen/qr/internal/application/input/ports"
	projectlotcreateduc "rutadeorigen/qr/internal/application/usecases/projectlotcreated"
)

var _ ports.ProjectLotCreatedPort = (*Adapter)(nil)

type Adapter struct {
	useCase   projectlotcreateduc.UseCase
	assembler assembler.DTOAssembler[projectlotcreateduc.Command, request.ProjectLotCreated, response.Empty]
}

func New(useCase projectlotcreateduc.UseCase) *Adapter {
	return &Adapter{useCase: useCase, assembler: Assembler{}}
}

func (a *Adapter) Execute(ctx context.Context, input request.ProjectLotCreated) (response.Empty, error) {
	cmd, err := a.assembler.ToDomain(input)
	if err != nil {
		return response.Empty{}, err
	}

	if err := a.useCase.Execute(ctx, cmd); err != nil {
		return response.Empty{}, err
	}

	return a.assembler.ToDTO(cmd), nil
}
