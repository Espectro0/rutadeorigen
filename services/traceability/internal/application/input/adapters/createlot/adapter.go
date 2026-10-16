package createlot

import (
	"context"

	"rutadeorigen/traceability/internal/application/input/assembler"
	"rutadeorigen/traceability/internal/application/input/dto/request"
	"rutadeorigen/traceability/internal/application/input/dto/response"
	"rutadeorigen/traceability/internal/application/input/ports"
	createlotuc "rutadeorigen/traceability/internal/application/usecases/createlot"
	"rutadeorigen/traceability/internal/domain/lot"
)

var _ ports.CreateLotPort = (*Adapter)(nil)

type Adapter struct {
	useCase   createlotuc.UseCase
	assembler assembler.DTOAssembler[*lot.Lot, request.CreateLot, response.Lot]
}

func New(useCase createlotuc.UseCase) *Adapter {
	return &Adapter{useCase: useCase, assembler: Assembler{}}
}

func (a *Adapter) Execute(ctx context.Context, input request.CreateLot) (response.Lot, error) {
	draft, err := a.assembler.ToDomain(input)
	if err != nil {
		return response.Lot{}, err
	}

	if err := a.useCase.Execute(ctx, draft); err != nil {
		return response.Lot{}, err
	}

	return a.assembler.ToDTO(draft), nil
}
