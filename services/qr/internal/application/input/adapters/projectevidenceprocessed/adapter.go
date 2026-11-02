package projectevidenceprocessed

import (
	"context"

	"rutadeorigen/qr/internal/application/input/assembler"
	"rutadeorigen/qr/internal/application/input/dto/request"
	"rutadeorigen/qr/internal/application/input/dto/response"
	"rutadeorigen/qr/internal/application/input/ports"
	projectevidenceprocesseduc "rutadeorigen/qr/internal/application/usecases/projectevidenceprocessed"
)

var _ ports.ProjectEvidenceProcessedPort = (*Adapter)(nil)

type Adapter struct {
	useCase   projectevidenceprocesseduc.UseCase
	assembler assembler.DTOAssembler[projectevidenceprocesseduc.Command, request.ProjectEvidenceProcessed, response.Empty]
}

func New(useCase projectevidenceprocesseduc.UseCase) *Adapter {
	return &Adapter{useCase: useCase, assembler: Assembler{}}
}

func (a *Adapter) Execute(ctx context.Context, input request.ProjectEvidenceProcessed) (response.Empty, error) {
	cmd, err := a.assembler.ToDomain(input)
	if err != nil {
		return response.Empty{}, err
	}

	if err := a.useCase.Execute(ctx, cmd); err != nil {
		return response.Empty{}, err
	}

	return a.assembler.ToDTO(cmd), nil
}
