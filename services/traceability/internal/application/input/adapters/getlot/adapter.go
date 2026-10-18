package getlot

import (
	"context"

	"rutadeorigen/traceability/internal/application/input/adapters/lotmapper"
	"rutadeorigen/traceability/internal/application/input/dto/request"
	"rutadeorigen/traceability/internal/application/input/dto/response"
	"rutadeorigen/traceability/internal/application/input/ports"
	getlotuc "rutadeorigen/traceability/internal/application/usecases/getlot"
	"rutadeorigen/traceability/internal/platform/exception"
	"rutadeorigen/traceability/internal/platform/helper"
)

var _ ports.GetLotPort = (*Adapter)(nil)

type Adapter struct {
	useCase getlotuc.UseCase
}

func New(useCase getlotuc.UseCase) *Adapter {
	return &Adapter{useCase: useCase}
}

func (a *Adapter) Execute(ctx context.Context, input request.GetLot) (response.Lot, error) {
	id := helper.ParseUUID(input.ID)
	if helper.IsDefaultUUID(id) {
		return response.Lot{}, exception.New(exception.CodeLotNotFound)
	}

	l, err := a.useCase.Execute(ctx, id)
	if err != nil {
		return response.Lot{}, err
	}
	return lotmapper.ToResponse(l), nil
}
