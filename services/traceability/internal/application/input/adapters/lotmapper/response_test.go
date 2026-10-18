package lotmapper_test

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"rutadeorigen/traceability/internal/application/input/adapters/lotmapper"
	"rutadeorigen/traceability/internal/application/input/dto/response"
	"rutadeorigen/traceability/internal/domain/lot"
)

func restoredLot(status lot.Status, voidReason string) (*lot.Lot, lot.RestoreParams) {
	params := lot.RestoreParams{
		ID:           uuid.New(),
		FarmID:       uuid.New(),
		CustodianID:  uuid.New(),
		Code:         "LOT-001",
		Variety:      "Castillo",
		Process:      lot.ProcessWashed,
		HarvestDate:  time.Date(2026, time.September, 15, 0, 0, 0, 0, time.UTC),
		CurrentStage: lot.StageRoasting,
		Status:       status,
		VoidReason:   voidReason,
		Version:      3,
		CreatedAt:    time.Date(2026, time.September, 20, 14, 30, 0, 0, time.UTC),
	}
	return lot.Restore(params), params
}

func TestToResponse(t *testing.T) {
	t.Run("maps every field", func(t *testing.T) {
		l, p := restoredLot(lot.StatusActive, "")

		got := lotmapper.ToResponse(l)

		want := response.Lot{
			ID:          p.ID.String(),
			Version:     p.Version,
			FarmID:      p.FarmID.String(),
			CustodianID: p.CustodianID.String(),
			Code:        "LOT-001",
			Variety:     "Castillo",
			Process:     "washed",
			HarvestDate: "2026-09-15",
			Stage:       "roasting",
			Status:      "active",
			VoidReason:  "",
			CreatedAt:   "2026-09-20T14:30:00Z",
		}
		if got != want {
			t.Errorf("got %+v, want %+v", got, want)
		}
	})

	t.Run("includes the void reason of a voided lot", func(t *testing.T) {
		l, _ := restoredLot(lot.StatusVoided, "wrong farm")

		got := lotmapper.ToResponse(l)

		if got.Status != "voided" {
			t.Errorf("Status: got %q, want %q", got.Status, "voided")
		}
		if got.VoidReason != "wrong farm" {
			t.Errorf("VoidReason: got %q, want %q", got.VoidReason, "wrong farm")
		}
	})

	t.Run("default lot returns an empty response", func(t *testing.T) {
		if got := lotmapper.ToResponse(lot.Default()); got != (response.Lot{}) {
			t.Errorf("got %+v, want empty response", got)
		}
	})

	t.Run("nil lot returns an empty response", func(t *testing.T) {
		if got := lotmapper.ToResponse(nil); got != (response.Lot{}) {
			t.Errorf("got %+v, want empty response", got)
		}
	})
}
