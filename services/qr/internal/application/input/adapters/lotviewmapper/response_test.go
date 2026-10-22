package lotviewmapper_test

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"rutadeorigen/qr/internal/application/input/adapters/lotviewmapper"
	"rutadeorigen/qr/internal/application/input/dto/response"
	"rutadeorigen/qr/internal/domain/lotview"
)

func restoredView(stage lotview.Stage, status lotview.Status, voidReason string) *lotview.LotView {
	return lotview.Restore(lotview.RestoreParams{
		LotID:        uuid.New(),
		Code:         "LT-001",
		Variety:      "Castillo",
		Process:      lotview.ProcessWashed,
		HarvestDate:  time.Date(2026, time.September, 15, 0, 0, 0, 0, time.UTC),
		FarmID:       uuid.New(),
		FarmName:     "La Esperanza",
		FarmAltitude: 1800,
		ProducerID:   uuid.New(),
		ProducerName: "Maria Lopez",
		CustodianID:  uuid.New(),
		CurrentStage: stage,
		Status:       status,
		VoidReason:   voidReason,
		Version:      3,
		CreatedAt:    time.Date(2026, time.September, 20, 14, 30, 0, 0, time.UTC),
		UpdatedAt:    time.Date(2026, time.September, 21, 8, 0, 0, 0, time.UTC),
	})
}

func TestToPublicLot(t *testing.T) {
	t.Run("maps every public field", func(t *testing.T) {
		v := restoredView(lotview.StagePackaging, lotview.StatusActive, "")

		got := lotviewmapper.ToPublicLot(v)

		want := response.PublicLot{
			ID:          v.LotID().String(),
			Code:        "LT-001",
			Variety:     "Castillo",
			Process:     "washed",
			HarvestDate: "2026-09-15",
			Farm:        response.PublicFarm{Name: "La Esperanza", Altitude: 1800},
			Producer:    response.PublicProducer{Name: "Maria Lopez"},
			Stage:       "packaging",
			Status:      "active",
			QRAvailable: true,
			UpdatedAt:   "2026-09-21T08:00:00Z",
		}
		if got != want {
			t.Errorf("got %+v, want %+v", got, want)
		}
	})

	t.Run("does not expose internal identifiers", func(t *testing.T) {
		v := restoredView(lotview.StagePackaging, lotview.StatusActive, "")

		got := lotviewmapper.ToPublicLot(v)

		if got.ID == v.CustodianID().String() || got.ID == v.FarmID().String() || got.ID == v.ProducerID().String() {
			t.Error("the ID must be the lot ID")
		}
	})

	t.Run("the QR is not available before packaging", func(t *testing.T) {
		for _, stage := range []lotview.Stage{lotview.StageNone, lotview.StageFarm, lotview.StageRoasting} {
			got := lotviewmapper.ToPublicLot(restoredView(stage, lotview.StatusActive, ""))

			if got.QRAvailable {
				t.Errorf("stage %s: the QR must not be available", stage)
			}
		}
	})

	t.Run("a voided lot carries its reason and has no QR", func(t *testing.T) {
		got := lotviewmapper.ToPublicLot(restoredView(lotview.StagePackaging, lotview.StatusVoided, "wrong batch"))

		if got.Status != "voided" || got.VoidReason != "wrong batch" || got.QRAvailable {
			t.Errorf("got %+v", got)
		}
	})

	t.Run("default and nil return an empty response", func(t *testing.T) {
		if got := lotviewmapper.ToPublicLot(lotview.Default()); got != (response.PublicLot{}) {
			t.Errorf("default: got %+v, want empty response", got)
		}
		if got := lotviewmapper.ToPublicLot(nil); got != (response.PublicLot{}) {
			t.Errorf("nil: got %+v, want empty response", got)
		}
	})
}
