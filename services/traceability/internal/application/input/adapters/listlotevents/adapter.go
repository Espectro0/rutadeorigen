package listlotevents

import (
	"context"

	"rutadeorigen/traceability/internal/application/input/adapters/lotmapper"
	"rutadeorigen/traceability/internal/application/input/dto/request"
	"rutadeorigen/traceability/internal/application/input/dto/response"
	"rutadeorigen/traceability/internal/application/input/ports"
	listlotevents "rutadeorigen/traceability/internal/application/usecases/listlotevents"
	"rutadeorigen/traceability/internal/platform/exception"
	"rutadeorigen/traceability/internal/platform/helper"
)

var _ ports.ListLotEventsPort = (*Adapter)(nil)

type Adapter struct {
	useCase listlotevents.UseCase
}

func New(useCase listlotevents.UseCase) *Adapter {
	return &Adapter{useCase: useCase}
}

func (a *Adapter) Execute(ctx context.Context, input request.ListLotEvents) (response.LotEventList, error) {
	id := helper.ParseUUID(input.LotID)
	if helper.IsDefaultUUID(id) {
		return response.LotEventList{}, exception.New(exception.CodeLotNotFound)
	}

	res, err := a.useCase.Execute(ctx, id)
	if err != nil {
		return response.LotEventList{}, err
	}

	items := make([]response.LotEvent, 0, len(res.Events))
	for _, e := range res.Events {
		items = append(items, lotmapper.ToEventResponse(e))
	}
	return response.LotEventList{Items: items, ChainValid: res.ChainValid}, nil
}
