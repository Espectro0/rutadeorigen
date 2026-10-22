package listfarm_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"rutadeorigen/traceability/internal/application/input/adapters/listfarm"
	"rutadeorigen/traceability/internal/application/input/dto/request"
	listfarmuc "rutadeorigen/traceability/internal/application/usecases/listfarm"
	"rutadeorigen/traceability/internal/domain/farm"
	"rutadeorigen/traceability/internal/platform/exception"
)

type ctxKey struct{}

type fakeUseCase struct {
	result []*farm.Farm
	err    error

	calls int
	ctx   context.Context
}

func (f *fakeUseCase) Execute(ctx context.Context, _ listfarmuc.Query) ([]*farm.Farm, error) {
	f.calls++
	f.ctx = ctx
	if f.err != nil {
		return []*farm.Farm{}, f.err
	}
	return f.result, nil
}

func restoredFarm(name string) *farm.Farm {
	return farm.Restore(uuid.New(), uuid.New(), name, 1650, time.Date(2026, time.September, 20, 14, 30, 0, 0, time.UTC))
}

func TestAdapterExecute(t *testing.T) {
	t.Run("returns the mapped farms of the use case", func(t *testing.T) {
		first, second := restoredFarm("La Esperanza"), restoredFarm("El Mirador")
		uc := &fakeUseCase{result: []*farm.Farm{first, second}}
		ctx := context.WithValue(context.Background(), ctxKey{}, "marker")

		got, err := listfarm.New(uc).Execute(ctx, request.ListFarms{})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if uc.calls != 1 {
			t.Fatalf("use case calls: got %d, want 1", uc.calls)
		}
		if uc.ctx.Value(ctxKey{}) != "marker" {
			t.Error("the adapter must pass the same context to the use case")
		}
		if len(got.Items) != 2 || got.Items[0].Name != "La Esperanza" || got.Items[1].Name != "El Mirador" {
			t.Errorf("unexpected items: %+v", got.Items)
		}
	})

	t.Run("no farms returns empty, non-nil items", func(t *testing.T) {
		uc := &fakeUseCase{result: []*farm.Farm{}}

		got, err := listfarm.New(uc).Execute(context.Background(), request.ListFarms{})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Items == nil || len(got.Items) != 0 {
			t.Errorf("got %+v, want empty, non-nil items", got.Items)
		}
	})

	useCaseErrors := []struct {
		name string
		err  error
	}{
		{"actor not found", exception.New(exception.CodeActorNotFound)},
		{"not allowed", exception.New(exception.CodeForbidden)},
		{"unexpected error", errors.New("boom")},
	}

	for _, tc := range useCaseErrors {
		t.Run("propagates "+tc.name+" unchanged", func(t *testing.T) {
			uc := &fakeUseCase{err: tc.err}

			got, err := listfarm.New(uc).Execute(context.Background(), request.ListFarms{})

			if !errors.Is(err, tc.err) {
				t.Fatalf("got error %v, want the same error %v", err, tc.err)
			}
			if got.Items == nil || len(got.Items) != 0 {
				t.Errorf("got %+v, want empty, non-nil items", got.Items)
			}
		})
	}
}
