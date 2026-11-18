package stageview_test

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"rutadeorigen/qr/internal/domain/lotview"
	"rutadeorigen/qr/internal/domain/stageview"
	"rutadeorigen/qr/internal/platform/exception"
)

func validParams() stageview.NewParams {
	return stageview.NewParams{
		LotID:     uuid.New(),
		Stage:     lotview.StageFarm,
		EventDate: time.Date(2026, 9, 19, 15, 30, 0, 0, time.UTC),
		Details: stageview.Details{
			DryingMethod: "  African_Bed ",
			DryingDays:   15,
		},
	}
}

func TestNew(t *testing.T) {
	t.Run("builds a stage view with normalized values", func(t *testing.T) {
		p := validParams()

		got, err := stageview.New(p)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.LotID() != p.LotID || got.Stage() != lotview.StageFarm {
			t.Errorf("unexpected identity: %s %q", got.LotID(), got.Stage())
		}
		wantDate := time.Date(2026, 9, 19, 0, 0, 0, 0, time.UTC)
		if !got.EventDate().Equal(wantDate) {
			t.Errorf("EventDate: got %v, want %v", got.EventDate(), wantDate)
		}
		if got.Details().DryingMethod != "african_bed" || got.Details().DryingDays != 15 {
			t.Errorf("unexpected details %+v", got.Details())
		}
		if got.IsDefault() {
			t.Error("a built stage view must not be the default")
		}
	})

	t.Run("discards negative numbers", func(t *testing.T) {
		p := validParams()
		p.Details = stageview.Details{DryingDays: -1, RoastDurationMinutes: -2, PackageWeightGrams: -3}

		got, err := stageview.New(p)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Details() != (stageview.Details{}) {
			t.Errorf("expected empty details, got %+v", got.Details())
		}
	})

	t.Run("rejects an invalid stage", func(t *testing.T) {
		for _, stage := range []lotview.Stage{lotview.StageNone, lotview.StageUnknown, lotview.Stage("drying")} {
			p := validParams()
			p.Stage = stage

			got, err := stageview.New(p)

			if !exception.HasCode(err, exception.CodeLotViewInvalidStage) {
				t.Errorf("stage %q: got %v, want %s", stage, err, exception.CodeLotViewInvalidStage)
			}
			if !got.IsDefault() {
				t.Errorf("stage %q: expected the default stage view", stage)
			}
		}
	})

	t.Run("rejects a default lot ID or event date", func(t *testing.T) {
		tests := map[string]func(p *stageview.NewParams){
			"default lot ID":     func(p *stageview.NewParams) { p.LotID = uuid.Nil },
			"default event date": func(p *stageview.NewParams) { p.EventDate = time.Time{} },
		}

		for name, mutate := range tests {
			p := validParams()
			mutate(&p)

			got, err := stageview.New(p)

			if !exception.HasCode(err, exception.CodeEventInvalidPayload) {
				t.Errorf("%s: got %v, want %s", name, err, exception.CodeEventInvalidPayload)
			}
			if !got.IsDefault() {
				t.Errorf("%s: expected the default stage view", name)
			}
		}
	})
}

func TestDefaultIsDefault(t *testing.T) {
	if !stageview.Default().IsDefault() {
		t.Error("the default stage view must report IsDefault")
	}
}
