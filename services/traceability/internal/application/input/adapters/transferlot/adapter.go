package transferlot

import (
	"context"

	"rutadeorigen/traceability/internal/application/input/dto/request"
	"rutadeorigen/traceability/internal/application/input/dto/response"
	"rutadeorigen/traceability/internal/application/input/ports"
	transferlotuc "rutadeorigen/traceability/internal/application/usecases/transferlot"
)

var _ ports.TransferLotPort = (*Adapter)(nil)

type Adapter struct {
	useCase   transferlotuc.UseCase
	assembler Assembler
}

func New(useCase transferlotuc.UseCase) *Adapter {
	return &Adapter{useCase: useCase, assembler: Assembler{}}
}

func (a *Adapter) Execute(ctx context.Context, input request.TransferLot) (response.TransferredLot, error) {
	cmd, err := a.assembler.ToCommand(input)
	if err != nil {
		return response.TransferredLot{}, err
	}

	res, err := a.useCase.Execute(ctx, cmd)
	if err != nil {
		return response.TransferredLot{}, err
	}

	return a.assembler.ToDTO(res), nil
}
