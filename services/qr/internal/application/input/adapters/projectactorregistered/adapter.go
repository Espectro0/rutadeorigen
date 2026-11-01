package projectactorregistered

import (
	"context"

	"rutadeorigen/qr/internal/application/input/assembler"
	"rutadeorigen/qr/internal/application/input/dto/request"
	"rutadeorigen/qr/internal/application/input/dto/response"
	"rutadeorigen/qr/internal/application/input/ports"
	projectactorregistereduc "rutadeorigen/qr/internal/application/usecases/projectactorregistered"
)

var _ ports.ProjectActorRegisteredPort = (*Adapter)(nil)

type Adapter struct {
	useCase   projectactorregistereduc.UseCase
	assembler assembler.DTOAssembler[projectactorregistereduc.Command, request.ProjectActorRegistered, response.Empty]
}

func New(useCase projectactorregistereduc.UseCase) *Adapter {
	return &Adapter{
		useCase:   useCase,
		assembler: Assembler{},
	}
}

func (a *Adapter) Execute(ctx context.Context, input request.ProjectActorRegistered) (response.Empty, error) {
	cmd, err := a.assembler.ToDomain(input)
	if err != nil {
		return response.Empty{}, err
	}

	if err := a.useCase.Execute(ctx, cmd); err != nil {
		return response.Empty{}, err
	}

	return a.assembler.ToDTO(cmd), nil
}
