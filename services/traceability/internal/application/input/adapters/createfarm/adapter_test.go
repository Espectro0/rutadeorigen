package createfarm_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"rutadeorigen/traceability/internal/application/input/adapters/createfarm"
	"rutadeorigen/traceability/internal/application/input/dto/response"
	"rutadeorigen/traceability/internal/domain/farm"
	"rutadeorigen/traceability/internal/platform/exception"
)

type ctxKey struct{}

type fakeUseCase struct {
	result *farm.Farm
	err    error

	calls    int
	received *farm.Farm
	ctx      context.Context
}

func (f *fakeUseCase) Execute(ctx context.Context, draft *farm.Farm) error {
	f.calls++
	received := *draft
	f.received = &received
	f.ctx = ctx
	if f.err != nil {
		return f.err
	}
	*draft = *f.result
	return nil
}

func createdFarm() *farm.Farm {
	return farm.Restore(uuid.New(), uuid.New(), "La Esperanza", 1650, time.Date(2026, time.September, 20, 14, 30, 0, 0, time.UTC))
}

func TestAdapterExecute(t *testing.T) {
	t.Run("sends the draft to the use case and returns the mapped farm", func(t *testing.T) {
		created := createdFarm()
		uc := &fakeUseCase{result: created}
		ctx := context.WithValue(context.Background(), ctxKey{}, "marker")

		got, err := createfarm.New(uc).Execute(ctx, validRequest())

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if uc.calls != 1 {
			t.Fatalf("use case calls: got %d, want 1", uc.calls)
		}
		if uc.received.Name() != "La Esperanza" || uc.received.Altitude() != 1650 {
			t.Errorf("use case received an unexpected draft: %q %d", uc.received.Name(), uc.received.Altitude())
		}
		if uc.ctx.Value(ctxKey{}) != "marker" {
			t.Error("the adapter must pass the same context to the use case")
		}
		if got.ID != created.ID().String() || got.ProducerID != created.ProducerID().String() {
			t.Errorf("unexpected response: %+v", got)
		}
		if got.CreatedAt != "2026-09-20T14:30:00Z" {
			t.Errorf("CreatedAt: got %q, want %q", got.CreatedAt, "2026-09-20T14:30:00Z")
		}
	})

	useCaseErrors := []struct {
		name string
		err  error
	}{
		{"actor not found", exception.New(exception.CodeActorNotFound)},
		{"not a producer", exception.New(exception.CodeFarmInvalidProducer)},
		{"not allowed", exception.New(exception.CodeForbidden)},
		{"invalid name", exception.New(exception.CodeFarmInvalidName)},
		{"invalid altitude", exception.New(exception.CodeFarmInvalidAltitude)},
		{"unexpected error", errors.New("boom")},
	}

	for _, tc := range useCaseErrors {
		t.Run("propagates "+tc.name+" unchanged", func(t *testing.T) {
			uc := &fakeUseCase{err: tc.err}

			got, err := createfarm.New(uc).Execute(context.Background(), validRequest())

			if !errors.Is(err, tc.err) {
				t.Fatalf("got error %v, want the same error %v", err, tc.err)
			}
			if got != (response.Farm{}) {
				t.Errorf("got %+v, want empty response", got)
			}
		})
	}
}
