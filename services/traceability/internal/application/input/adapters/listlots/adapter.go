package listlots

import (
	"context"

	"rutadeorigen/traceability/internal/application/input/adapters/lotmapper"
	"rutadeorigen/traceability/internal/application/input/dto/request"
	"rutadeorigen/traceability/internal/application/input/dto/response"
	"rutadeorigen/traceability/internal/application/input/ports"
	listlotsuc "rutadeorigen/traceability/internal/application/usecases/listlots"
	"rutadeorigen/traceability/internal/domain/lot"
)

var _ ports.ListLotsPort = (*Adapter)(nil)

type Adapter struct {
	useCase listlotsuc.UseCase
}

func New(useCase listlotsuc.UseCase) *Adapter {
	return &Adapter{useCase: useCase}
}

func (a *Adapter) Execute(ctx context.Context, input request.ListLots) (response.LotPage, error) {
	query, err := toQuery(input)
	if err != nil {
		return response.LotPage{}, err
	}

	page, err := a.useCase.Execute(ctx, query)
	if err != nil {
		return response.LotPage{}, err
	}
	return toPage(page), nil
}

func toPage(p lot.Page) response.LotPage {
	items := make([]response.Lot, 0, len(p.Lots))
	for _, l := range p.Lots {
		items = append(items, lotmapper.ToResponse(l))
	}
	return response.LotPage{Items: items, NextCursor: encodeCursor(p.Next)}
}
