package getpubliclot

import (
	"context"

	"rutadeorigen/qr/internal/application/input/adapters/lotviewmapper"
	"rutadeorigen/qr/internal/application/input/dto/request"
	"rutadeorigen/qr/internal/application/input/dto/response"
	"rutadeorigen/qr/internal/application/input/ports"
	getpubliclotuc "rutadeorigen/qr/internal/application/usecases/getpubliclot"
	listpublicevidencesuc "rutadeorigen/qr/internal/application/usecases/listpublicevidences"
	"rutadeorigen/qr/internal/domain/lotview"
	"rutadeorigen/qr/internal/platform/exception"
	"rutadeorigen/qr/internal/platform/helper"
)

var _ ports.GetPublicLotPort = (*Adapter)(nil)

type withoutEvidences struct{}

func (withoutEvidences) Execute(context.Context, *lotview.LotView) ([]listpublicevidencesuc.Item, error) {
	return []listpublicevidencesuc.Item{}, nil
}

type Adapter struct {
	useCase   getpubliclotuc.UseCase
	evidences listpublicevidencesuc.UseCase
}

func New(useCase getpubliclotuc.UseCase) *Adapter {
	return &Adapter{useCase: useCase, evidences: withoutEvidences{}}
}

func (a *Adapter) WithEvidences(evidences listpublicevidencesuc.UseCase) *Adapter {
	a.evidences = evidences
	return a
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

	items, err := a.evidences.Execute(ctx, view)
	if err != nil {
		return response.PublicLot{}, err
	}

	lot := lotviewmapper.ToPublicLot(view)
	lot.Evidences = toPublicEvidences(items)
	return lot, nil
}

func toPublicEvidences(items []listpublicevidencesuc.Item) []response.PublicEvidence {
	evidences := make([]response.PublicEvidence, 0, len(items))
	for _, item := range items {
		evidences = append(evidences, response.PublicEvidence{
			ID:           item.Evidence.EvidenceID().String(),
			Stage:        item.Evidence.Stage().String(),
			ThumbnailURL: item.ThumbnailURL,
			ImageURL:     item.ImageURL,
		})
	}
	return evidences
}
