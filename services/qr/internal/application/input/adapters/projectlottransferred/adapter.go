package projectlottransferred

import (
	"context"

	"rutadeorigen/qr/internal/application/input/assembler"
	"rutadeorigen/qr/internal/application/input/dto/request"
	"rutadeorigen/qr/internal/application/input/dto/response"
	"rutadeorigen/qr/internal/application/input/ports"
	projectlottransferreduc "rutadeorigen/qr/internal/application/usecases/projectlottransferred"
)

var _ ports.ProjectLotTransferredPort = (*Adapter)(nil)

type Adapter struct {
	useCase   projectlottransferreduc.UseCase
	assembler assembler.DTOAssembler[projectlottransferreduc.Command, request.ProjectLotTransferred, response.Empty]
}

func New(useCase projectlottransferreduc.UseCase) *Adapter {
	return &Adapter{
		useCase:   useCase,
		assembler: Assembler{},
	}
}

func (a *Adapter) Execute(ctx context.Context, input request.ProjectLotTransferred) (response.Empty, error) {
	cmd, err := a.assembler.ToDomain(input)
	if err != nil {
		return response.Empty{}, err
	}

	if err := a.useCase.Execute(ctx, cmd); err != nil {
		return response.Empty{}, err
	}

	return a.assembler.ToDTO(cmd), nil
}
