package resolvescan

import (
	"context"

	"rutadeorigen/qr/internal/application/input/dto/request"
	"rutadeorigen/qr/internal/application/input/dto/response"
	"rutadeorigen/qr/internal/application/input/ports"
	resolvescanuc "rutadeorigen/qr/internal/application/usecases/resolvescan"
	"rutadeorigen/qr/internal/platform/helper"
)

var _ ports.ResolveScanPort = (*Adapter)(nil)

type Adapter struct {
	useCase resolvescanuc.UseCase
}

func New(useCase resolvescanuc.UseCase) *Adapter {
	return &Adapter{useCase: useCase}
}

func (a *Adapter) Execute(ctx context.Context, input request.ResolveScan) (response.ScanRedirect, error) {
	resolution, err := a.useCase.Execute(ctx, helper.ApplyTrim(input.Path))
	if err != nil {
		return response.ScanRedirect{}, err
	}

	return response.ScanRedirect{
		Location:   resolution.Location,
		StatusCode: resolution.StatusCode,
	}, nil
}
