package getpubliclot_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"rutadeorigen/qr/internal/application/input/adapters/getpubliclot"
	"rutadeorigen/qr/internal/application/input/dto/request"
	"rutadeorigen/qr/internal/domain/lotview"
	"rutadeorigen/qr/internal/platform/exception"
)

type fakeUseCase struct {
	result *lotview.LotView
	err    error

	calls    int
	received uuid.UUID
}

func (f *fakeUseCase) Execute(_ context.Context, id uuid.UUID) (*lotview.LotView, error) {
	f.calls++
	f.received = id
	if f.err != nil {
		return lotview.Default(), f.err
	}
	return f.result, nil
}

func existingView() *lotview.LotView {
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
		CurrentStage: lotview.StageRoasting,
		Status:       lotview.StatusActive,
		Version:      2,
		UpdatedAt:    time.Date(2026, time.September, 21, 8, 0, 0, 0, time.UTC),
	})
}

func TestAdapterExecute(t *testing.T) {
	ctx := context.Background()

	t.Run("sends the parsed id to the use case and maps the lot", func(t *testing.T) {
		v := existingView()
		uc := &fakeUseCase{result: v}

		got, err := getpubliclot.New(uc).Execute(ctx, request.GetPublicLot{ID: " " + v.LotID().String() + " "})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if uc.received != v.LotID() {
			t.Errorf("use case received %s, want %s", uc.received, v.LotID())
		}
		if got.ID != v.LotID().String() || got.Code != "LT-001" || got.Farm.Name != "La Esperanza" || got.Stage != "roasting" {
			t.Errorf("got %+v", got)
		}
	})

	t.Run("reports a malformed id as not found without calling the use case", func(t *testing.T) {
		for _, id := range []string{"", "not-a-uuid", "00000000-0000-0000-0000-000000000000"} {
			uc := &fakeUseCase{result: existingView()}

			_, err := getpubliclot.New(uc).Execute(ctx, request.GetPublicLot{ID: id})

			if !exception.HasCode(err, exception.CodeLotViewNotFound) {
				t.Errorf("id %q: got %v, want %s", id, err, exception.CodeLotViewNotFound)
			}
			if uc.calls != 0 {
				t.Errorf("id %q: use case calls: got %d, want 0", id, uc.calls)
			}
		}
	})

	t.Run("returns the use case error and an empty response", func(t *testing.T) {
		boom := errors.New("boom")
		uc := &fakeUseCase{err: boom}

		got, err := getpubliclot.New(uc).Execute(ctx, request.GetPublicLot{ID: uuid.NewString()})

		if !errors.Is(err, boom) {
			t.Errorf("got %v, want %v", err, boom)
		}
		if got.ID != "" {
			t.Errorf("got %+v, want an empty response", got)
		}
	})
}
