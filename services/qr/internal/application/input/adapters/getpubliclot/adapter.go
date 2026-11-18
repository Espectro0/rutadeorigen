package getpubliclot

import (
	"context"
	"time"

	"rutadeorigen/qr/internal/application/input/adapters/lotviewmapper"
	"rutadeorigen/qr/internal/application/input/dto/request"
	"rutadeorigen/qr/internal/application/input/dto/response"
	"rutadeorigen/qr/internal/application/input/ports"
	getpubliclotuc "rutadeorigen/qr/internal/application/usecases/getpubliclot"
	listpublicevidencesuc "rutadeorigen/qr/internal/application/usecases/listpublicevidences"
	listpublicstagesuc "rutadeorigen/qr/internal/application/usecases/listpublicstages"
	"rutadeorigen/qr/internal/domain/lotview"
	"rutadeorigen/qr/internal/domain/stageview"
	"rutadeorigen/qr/internal/platform/exception"
	"rutadeorigen/qr/internal/platform/helper"
)

var _ ports.GetPublicLotPort = (*Adapter)(nil)

type withoutEvidences struct{}

func (withoutEvidences) Execute(context.Context, *lotview.LotView) ([]listpublicevidencesuc.Item, error) {
	return []listpublicevidencesuc.Item{}, nil
}

type withoutStages struct{}

func (withoutStages) Execute(context.Context, *lotview.LotView) ([]*stageview.StageView, error) {
	return []*stageview.StageView{}, nil
}

type Adapter struct {
	useCase   getpubliclotuc.UseCase
	evidences listpublicevidencesuc.UseCase
	stages    listpublicstagesuc.UseCase
}

func New(useCase getpubliclotuc.UseCase) *Adapter {
	return &Adapter{useCase: useCase, evidences: withoutEvidences{}, stages: withoutStages{}}
}

func (a *Adapter) WithStages(stages listpublicstagesuc.UseCase) *Adapter {
	a.stages = stages
	return a
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

	stages, err := a.stages.Execute(ctx, view)
	if err != nil {
		return response.PublicLot{}, err
	}

	lot := lotviewmapper.ToPublicLot(view)
	lot.Evidences = toPublicEvidences(items)
	lot.Stages = toPublicStages(stages)
	return lot, nil
}

func toPublicStages(views []*stageview.StageView) []response.PublicStage {
	stages := make([]response.PublicStage, 0, len(views))
	for _, view := range views {
		details := view.Details()
		stages = append(stages, response.PublicStage{
			Stage:                view.Stage().String(),
			EventDate:            view.EventDate().Format(time.DateOnly),
			DryingMethod:         details.DryingMethod,
			DryingDays:           details.DryingDays,
			RoastLevel:           details.RoastLevel,
			RoastDurationMinutes: details.RoastDurationMinutes,
			Grind:                details.Grind,
			PackageWeightGrams:   details.PackageWeightGrams,
			BestBefore:           details.BestBefore,
		})
	}
	return stages
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
