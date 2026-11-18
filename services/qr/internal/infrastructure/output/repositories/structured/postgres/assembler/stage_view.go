package assembler

import (
	"rutadeorigen/qr/internal/domain/lotview"
	"rutadeorigen/qr/internal/domain/stageview"
	"rutadeorigen/qr/internal/infrastructure/output/repositories/structured/postgres/entity"
)

var _ EntityAssembler[entity.StageView, *stageview.StageView] = StageView{}

type StageView struct{}

func (StageView) ToDomain(e entity.StageView) *stageview.StageView {
	return stageview.Restore(stageview.RestoreParams{
		LotID:     e.LotID,
		Stage:     lotview.ParseStage(e.Stage),
		EventDate: e.EventDate,
		Details: stageview.Details{
			DryingMethod:         e.DryingMethod,
			DryingDays:           e.DryingDays,
			RoastLevel:           e.RoastLevel,
			RoastDurationMinutes: e.RoastDurationMinutes,
			Grind:                e.Grind,
			PackageWeightGrams:   e.PackageWeightGrams,
			BestBefore:           e.BestBefore,
		},
	})
}

func (StageView) ToEntity(v *stageview.StageView) entity.StageView {
	if v.IsDefault() {
		return entity.StageView{}
	}

	details := v.Details()
	return entity.StageView{
		LotID:                v.LotID(),
		Stage:                v.Stage().String(),
		EventDate:            v.EventDate().UTC(),
		DryingMethod:         details.DryingMethod,
		DryingDays:           details.DryingDays,
		RoastLevel:           details.RoastLevel,
		RoastDurationMinutes: details.RoastDurationMinutes,
		Grind:                details.Grind,
		PackageWeightGrams:   details.PackageWeightGrams,
		BestBefore:           details.BestBefore,
	}
}
