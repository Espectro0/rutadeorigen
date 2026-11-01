package lotview_test

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"rutadeorigen/qr/internal/domain/lotview"
	"rutadeorigen/qr/internal/platform/exception"
)

func TestApplyTransfer(t *testing.T) {
	at := time.Date(2026, 10, 2, 9, 0, 0, 0, time.UTC)

	t.Run("changes only the custodian, version and update time", func(t *testing.T) {
		view := applyStageView(t, nil)
		next := uuid.New()

		got, err := view.ApplyTransfer(next, 3, at)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.CustodianID() != next || got.Version() != 3 || !got.UpdatedAt().Equal(at) {
			t.Errorf("got custodian %s, version %d, updated at %v", got.CustodianID(), got.Version(), got.UpdatedAt())
		}
		if got.CurrentStage() != view.CurrentStage() || got.Status() != view.Status() || got.LotID() != view.LotID() {
			t.Errorf("unexpected changes: got %+v, original %+v", got, view)
		}
		if view.Version() != 2 || view.CustodianID() == next {
			t.Error("the original must not change")
		}
	})

	t.Run("uses UTC and the current time when none is given", func(t *testing.T) {
		view := applyStageView(t, nil)
		bogota := time.FixedZone("UTC-5", -5*60*60)

		inBogota, _ := view.ApplyTransfer(uuid.New(), 3, at.In(bogota))
		now, _ := view.ApplyTransfer(uuid.New(), 3, time.Time{})

		if inBogota.UpdatedAt().Location() != time.UTC || !inBogota.UpdatedAt().Equal(at) {
			t.Errorf("got %v, want %v in UTC", inBogota.UpdatedAt(), at)
		}
		if time.Since(now.UpdatedAt()) > time.Minute {
			t.Errorf("unexpected update time %v", now.UpdatedAt())
		}
	})

	t.Run("rejects invalid transfers", func(t *testing.T) {
		voided := applyStageView(t, func(p *lotview.NewParams) { p.Status = lotview.StatusVoided })
		tests := []struct {
			name    string
			view    *lotview.LotView
			target  uuid.UUID
			version int
			code    exception.Code
		}{
			{"default view", lotview.Default(), uuid.New(), 3, exception.CodeLotViewNotFound},
			{"no custodian", applyStageView(t, nil), uuid.Nil, 3, exception.CodeLotViewInvalidCustodian},
			{"voided lot", voided, uuid.New(), 3, exception.CodeLotViewVoided},
			{"same version", applyStageView(t, nil), uuid.New(), 2, exception.CodeLotViewInvalidVersion},
			{"older version", applyStageView(t, nil), uuid.New(), 1, exception.CodeLotViewInvalidVersion},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got, err := tt.view.ApplyTransfer(tt.target, tt.version, at)

				if !exception.HasCode(err, tt.code) || !got.IsDefault() {
					t.Errorf("got %+v and %v, want the default view and %s", got, err, tt.code)
				}
			})
		}
	})
}

func TestApplyVoid(t *testing.T) {
	at := time.Date(2026, 10, 2, 9, 0, 0, 0, time.UTC)

	t.Run("voids the lot with a trimmed reason", func(t *testing.T) {
		view := applyStageView(t, nil)

		got, err := view.ApplyVoid("  Lote contaminado ", 3, at)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Status() != lotview.StatusVoided || got.VoidReason() != "Lote contaminado" || got.Version() != 3 || !got.UpdatedAt().Equal(at) {
			t.Errorf("unexpected view %+v", got)
		}
		if got.CurrentStage() != view.CurrentStage() || got.CustodianID() != view.CustodianID() {
			t.Error("stage and custodian must not change")
		}
		if !view.IsActive() || view.VoidReason() != "" || view.Version() != 2 {
			t.Error("the original must not change")
		}
	})

	t.Run("rejects invalid voids", func(t *testing.T) {
		tests := []struct {
			name    string
			view    *lotview.LotView
			version int
			code    exception.Code
		}{
			{"default view", lotview.Default(), 3, exception.CodeLotViewNotFound},
			{"already voided", applyStageView(t, func(p *lotview.NewParams) { p.Status = lotview.StatusVoided }), 3, exception.CodeLotViewVoided},
			{"same version", applyStageView(t, nil), 2, exception.CodeLotViewInvalidVersion},
			{"older version", applyStageView(t, nil), 1, exception.CodeLotViewInvalidVersion},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got, err := tt.view.ApplyVoid("motivo", tt.version, at)

				if !exception.HasCode(err, tt.code) || !got.IsDefault() {
					t.Errorf("got %+v and %v, want the default view and %s", got, err, tt.code)
				}
			})
		}
	})
}
