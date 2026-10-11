package createlot_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"rutadeorigen/traceability/internal/application/input/adapters/createlot"
	"rutadeorigen/traceability/internal/application/input/dto/response"
	"rutadeorigen/traceability/internal/application/usecases"
	"rutadeorigen/traceability/internal/domain/lot"
)

type ctxKey struct{}

type fakeUseCase struct {
	result *lot.Lot
	err    error

	calls    int
	received *lot.Lot
	ctx      context.Context
}

func (f *fakeUseCase) Execute(ctx context.Context, draft *lot.Lot) (*lot.Lot, error) {
	f.calls++
	f.received = draft
	f.ctx = ctx
	if f.err != nil {
		return lot.Default(), f.err
	}
	return f.result, nil
}

func createdLot(farmID uuid.UUID) *lot.Lot {
	return lot.Restore(lot.RestoreParams{
		ID:           uuid.New(),
		FarmID:       farmID,
		CustodianID:  uuid.New(),
		Code:         "LOT-001",
		Variety:      "Castillo",
		Process:      lot.ProcessWashed,
		HarvestDate:  time.Date(2026, time.September, 15, 0, 0, 0, 0, time.UTC),
		CurrentStage: lot.StageNone,
		Status:       lot.StatusActive,
		Version:      1,
		CreatedAt:    time.Date(2026, time.September, 20, 14, 30, 0, 0, time.UTC),
	})
}

func TestAdapterExecute(t *testing.T) {
	t.Run("sends the draft to the use case and returns the mapped lot", func(t *testing.T) {
		farmID := uuid.New()
		created := createdLot(farmID)
		uc := &fakeUseCase{result: created}
		ctx := context.WithValue(context.Background(), ctxKey{}, "marker")

		got, err := createlot.New(uc).Execute(ctx, validRequest(farmID))

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if uc.calls != 1 {
			t.Fatalf("use case calls: got %d, want 1", uc.calls)
		}
		if uc.received.FarmID() != farmID || uc.received.Process() != lot.ProcessWashed {
			t.Errorf("use case received an unexpected draft: farm %s, process %q", uc.received.FarmID(), uc.received.Process())
		}
		if uc.ctx.Value(ctxKey{}) != "marker" {
			t.Error("the adapter must pass the same context to the use case")
		}
		if got.ID != created.ID().String() {
			t.Errorf("ID: got %q, want %q", got.ID, created.ID().String())
		}
		if got.CustodianID != created.CustodianID().String() {
			t.Errorf("CustodianID: got %q, want %q", got.CustodianID, created.CustodianID().String())
		}
		if got.Status != "active" || got.Stage != "none" {
			t.Errorf("Status/Stage: got %q/%q, want active/none", got.Status, got.Stage)
		}
	})

	t.Run("invalid request does not reach the use case", func(t *testing.T) {
		uc := &fakeUseCase{}
		req := validRequest(uuid.New())
		req.FarmID = "bad"

		got, err := createlot.New(uc).Execute(context.Background(), req)

		if !errors.Is(err, createlot.ErrInvalidFarmID) {
			t.Fatalf("got error %v, want %v", err, createlot.ErrInvalidFarmID)
		}
		if uc.calls != 0 {
			t.Errorf("use case calls: got %d, want 0", uc.calls)
		}
		if got != (response.Lot{}) {
			t.Errorf("got %+v, want empty response", got)
		}
	})

	useCaseErrors := []struct {
		name string
		err  error
	}{
		{"actor not found", usecases.ErrActorNotFound},
		{"farm not found", usecases.ErrFarmNotFound},
		{"not allowed", usecases.ErrNotAllowed},
		{"code already exists", usecases.ErrLotCodeAlreadyExists},
		{"unexpected error", errors.New("boom")},
	}

	for _, tc := range useCaseErrors {
		t.Run("propagates "+tc.name+" unchanged", func(t *testing.T) {
			uc := &fakeUseCase{err: tc.err}

			got, err := createlot.New(uc).Execute(context.Background(), validRequest(uuid.New()))

			if !errors.Is(err, tc.err) {
				t.Fatalf("got error %v, want the same error %v", err, tc.err)
			}
			if got != (response.Lot{}) {
				t.Errorf("got %+v, want empty response", got)
			}
		})
	}
}
