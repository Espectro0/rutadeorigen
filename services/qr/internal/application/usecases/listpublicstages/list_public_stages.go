package listpublicstages

import (
	"context"

	"rutadeorigen/qr/internal/application/output/ports"
	"rutadeorigen/qr/internal/domain/lotview"
	"rutadeorigen/qr/internal/domain/stageview"
	"rutadeorigen/qr/internal/platform/exception"
)

var _ UseCase = (*ListPublicStages)(nil)

type ListPublicStages struct {
	stages ports.StageViewRepository
}

func New(stages ports.StageViewRepository) *ListPublicStages {
	return &ListPublicStages{stages: stages}
}

func (uc *ListPublicStages) Execute(ctx context.Context, view *lotview.LotView) ([]*stageview.StageView, error) {
	if view.IsDefault() || !view.IsActive() {
		return []*stageview.StageView{}, nil
	}

	stored, err := uc.stages.ListByLotID(ctx, view.LotID())
	if err != nil {
		return []*stageview.StageView{}, exception.Wrap(err, "list stage views")
	}
	return stored, nil
}
