package getlot_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"rutadeorigen/traceability/internal/application/input/adapters/getlot"
	"rutadeorigen/traceability/internal/application/input/dto/request"
	"rutadeorigen/traceability/internal/domain/lot"
	"rutadeorigen/traceability/internal/platform/exception"
)

type fakeUseCase struct {
	result *lot.Lot
	err    error

	calls    int
	received uuid.UUID
}

func (f *fakeUseCase) Execute(_ context.Context, id uuid.UUID) (*lot.Lot, error) {
	f.calls++
	f.received = id
	if f.err != nil {
		return lot.Default(), f.err
	}
	return f.result, nil
}

func existingLot() *lot.Lot {
	return lot.Restore(lot.RestoreParams{
		ID:           uuid.New(),
		FarmID:       uuid.New(),
		CustodianID:  uuid.New(),
		Code:         "LOT-001",
		Variety:      "Castillo",
		Process:      lot.ProcessWashed,
		HarvestDate:  time.Date(2026, time.September, 15, 0, 0, 0, 0, time.UTC),
		CurrentStage: lot.StageRoasting,
		Status:       lot.StatusActive,
		Version:      4,
		CreatedAt:    time.Date(2026, time.September, 20, 14, 30, 0, 0, time.UTC),
	})
}

func TestAdapterExecute(t *testing.T) {
	ctx := context.Background()

	t.Run("sends the parsed id to the use case and maps the lot", func(t *testing.T) {
		l := existingLot()
		uc := &fakeUseCase{result: l}

		got, err := getlot.New(uc).Execute(ctx, request.GetLot{ID: " " + l.ID().String() + " "})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if uc.received != l.ID() {
			t.Errorf("use case received %s, want %s", uc.received, l.ID())
		}
		if got.ID != l.ID().String() || got.Code != "LOT-001" || got.Version != 4 || got.Stage != "roasting" {
			t.Errorf("got %+v", got)
		}
	})

	t.Run("reports a malformed id as not found without calling the use case", func(t *testing.T) {
		for _, id := range []string{"", "not-a-uuid", "00000000-0000-0000-0000-000000000000"} {
			uc := &fakeUseCase{result: existingLot()}

			_, err := getlot.New(uc).Execute(ctx, request.GetLot{ID: id})

			if !exception.HasCode(err, exception.CodeLotNotFound) {
				t.Errorf("id %q: got %v, want %s", id, err, exception.CodeLotNotFound)
			}
			if uc.calls != 0 {
				t.Errorf("id %q: use case calls: got %d, want 0", id, uc.calls)
			}
		}
	})

	t.Run("returns the use case error and an empty response", func(t *testing.T) {
		boom := errors.New("boom")
		uc := &fakeUseCase{err: boom}

		got, err := getlot.New(uc).Execute(ctx, request.GetLot{ID: uuid.NewString()})

		if !errors.Is(err, boom) {
			t.Errorf("got %v, want %v", err, boom)
		}
		if got.ID != "" {
			t.Errorf("got %+v, want an empty response", got)
		}
	})
}
