package getscanstats

import (
	"context"

	"rutadeorigen/qr/internal/application/input/dto/request"
	"rutadeorigen/qr/internal/application/input/dto/response"
	"rutadeorigen/qr/internal/application/input/ports"
	getscanstatsuc "rutadeorigen/qr/internal/application/usecases/getscanstats"
)

var _ ports.GetScanStatsPort = (*Adapter)(nil)

type Adapter struct {
	useCase   getscanstatsuc.UseCase
	assembler Assembler
}

func New(useCase getscanstatsuc.UseCase) *Adapter {
	return &Adapter{useCase: useCase, assembler: Assembler{}}
}

func (a *Adapter) Execute(ctx context.Context, input request.GetScanStats) (response.ScanStats, error) {
	query, err := a.assembler.ToQuery(input)
	if err != nil {
		return response.ScanStats{}, err
	}

	stats, err := a.useCase.Execute(ctx, query)
	if err != nil {
		return response.ScanStats{}, err
	}

	return a.assembler.ToDTO(stats, query.Days), nil
}
