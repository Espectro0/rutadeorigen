package voidlot

import (
	"context"

	"rutadeorigen/traceability/internal/application/input/adapters/lotmapper"
	"rutadeorigen/traceability/internal/application/input/dto/request"
	"rutadeorigen/traceability/internal/application/input/dto/response"
	"rutadeorigen/traceability/internal/application/input/ports"
	voidlotuc "rutadeorigen/traceability/internal/application/usecases/voidlot"
)

var _ ports.VoidLotPort = (*Adapter)(nil)

type Adapter struct {
	useCase   voidlotuc.UseCase
	assembler Assembler
}

func New(useCase voidlotuc.UseCase) *Adapter {
	return &Adapter{useCase: useCase, assembler: Assembler{}}
}

func (a *Adapter) Execute(ctx context.Context, input request.VoidLot) (response.Lot, error) {
	cmd, err := a.assembler.ToCommand(input)
	if err != nil {
		return response.Lot{}, err
	}

	l, err := a.useCase.Execute(ctx, cmd)
	if err != nil {
		return response.Lot{}, err
	}
	return lotmapper.ToResponse(l), nil
}
