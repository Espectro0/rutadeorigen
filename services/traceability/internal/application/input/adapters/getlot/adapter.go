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

func (a *Adapter) Execute(ctx context.Context, input request.GetLot) (response.LotDetail, error) {
	id := helper.ParseUUID(input.ID)
	if helper.IsDefaultUUID(id) {
		return response.LotDetail{}, exception.New(exception.CodeLotNotFound)
	}

	res, err := a.useCase.Execute(ctx, id)
	if err != nil {
		return response.LotDetail{}, err
	}

	actions := make([]string, 0, len(res.AllowedActions))
	for _, action := range res.AllowedActions {
		actions = append(actions, string(action))
	}
	return response.LotDetail{Lot: lotmapper.ToResponse(res.Lot), AllowedActions: actions}, nil
}
