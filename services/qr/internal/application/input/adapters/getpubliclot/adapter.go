package getpubliclot

import (
	"context"

	"rutadeorigen/qr/internal/application/input/adapters/lotviewmapper"
	"rutadeorigen/qr/internal/application/input/dto/request"
	"rutadeorigen/qr/internal/application/input/dto/response"
	"rutadeorigen/qr/internal/application/input/ports"
	getpubliclotuc "rutadeorigen/qr/internal/application/usecases/getpubliclot"
	"rutadeorigen/qr/internal/platform/exception"
	"rutadeorigen/qr/internal/platform/helper"
)

var _ ports.GetPublicLotPort = (*Adapter)(nil)

type Adapter struct {
	useCase getpubliclotuc.UseCase
}

func New(useCase getpubliclotuc.UseCase) *Adapter {
	return &Adapter{useCase: useCase}
}

func (a *Adapter) Execute(ctx context.Context, input request.GetPublicLot) (response.PublicLot, error) {
	id := helper.ParseUUID(input.ID)
	if helper.IsDefaultUUID(id) {
		return response.PublicLot{}, exception.New(exception.CodeLotViewNotFound)
	}

	view, err := a.useCase.Execute(ctx, id)
	if err != nil {
		return response.PublicLot{}, err
	}
	return lotviewmapper.ToPublicLot(view), nil
}
