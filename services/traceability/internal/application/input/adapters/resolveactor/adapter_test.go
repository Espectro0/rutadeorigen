package resolveactor_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"rutadeorigen/lib/requestcontext"
	"rutadeorigen/traceability/internal/application/input/adapters/resolveactor"
	"rutadeorigen/traceability/internal/application/input/dto/request"
	"rutadeorigen/traceability/internal/application/input/dto/response"
	"rutadeorigen/traceability/internal/domain/actor"
	"rutadeorigen/traceability/internal/platform/exception"
)

type ctxKey struct{}

type fakeUseCase struct {
	result *actor.Actor
	err    error

	calls    int
	received string
	ctx      context.Context
}

func (f *fakeUseCase) Execute(ctx context.Context, userID string) (*actor.Actor, error) {
	f.calls++
	f.received = userID
	f.ctx = ctx
	if f.err != nil {
		return actor.Default(), f.err
	}
	return f.result, nil
}

func restoredActor() *actor.Actor {
	return actor.Restore(uuid.New(), "user_01", actor.TypeProducer, "Finca La Esperanza SAS", "producer@example.com", actor.StatusActive, time.Date(2026, time.September, 20, 14, 30, 0, 0, time.UTC))
}

func TestAdapterExecute(t *testing.T) {
	t.Run("returns the id of the resolved actor", func(t *testing.T) {
		stored := restoredActor()
		uc := &fakeUseCase{result: stored}
		ctx := requestcontext.WithUserID(context.Background(), "user_01")

		got, err := resolveactor.New(uc).Execute(ctx, request.ResolveActor{})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.ID != stored.ID() {
			t.Errorf("got %s, want %s", got.ID, stored.ID())
		}
	})

	t.Run("passes the session user id to the use case", func(t *testing.T) {
		uc := &fakeUseCase{result: restoredActor()}
		ctx := requestcontext.WithUserID(context.Background(), "user_01")

		if _, err := resolveactor.New(uc).Execute(ctx, request.ResolveActor{}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if uc.calls != 1 || uc.received != "user_01" {
			t.Errorf("got %d calls with %q, want 1 call with user_01", uc.calls, uc.received)
		}
	})

	t.Run("passes the same context to the use case", func(t *testing.T) {
		uc := &fakeUseCase{result: restoredActor()}
		ctx := context.WithValue(context.Background(), ctxKey{}, "marker")

		if _, err := resolveactor.New(uc).Execute(ctx, request.ResolveActor{}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if uc.ctx.Value(ctxKey{}) != "marker" {
			t.Error("the adapter must pass the same context to the use case")
		}
	})

	t.Run("without a user id in the context passes an empty one", func(t *testing.T) {
		uc := &fakeUseCase{err: exception.New(exception.CodeUnauthorized)}

		_, err := resolveactor.New(uc).Execute(context.Background(), request.ResolveActor{})

		if uc.received != "" {
			t.Errorf("received %q, want empty", uc.received)
		}
		if !exception.HasCode(err, exception.CodeUnauthorized) {
			t.Errorf("got error %v, want unauthorized", err)
		}
	})

	useCaseErrors := []struct {
		name string
		err  error
	}{
		{"unauthorized", exception.New(exception.CodeUnauthorized)},
		{"actor not found", exception.New(exception.CodeActorNotFound)},
		{"forbidden", exception.New(exception.CodeForbidden)},
		{"internal", exception.New(exception.CodeInternal).WithTechnical("boom")},
		{"plain error", errors.New("boom")},
	}

	for _, tt := range useCaseErrors {
		t.Run("propagates "+tt.name, func(t *testing.T) {
			uc := &fakeUseCase{err: tt.err}

			got, err := resolveactor.New(uc).Execute(context.Background(), request.ResolveActor{})

			if !errors.Is(err, tt.err) {
				t.Errorf("got error %v, want %v", err, tt.err)
			}
			if got != (response.ResolvedActor{}) {
				t.Errorf("got %+v, want empty response", got)
			}
		})
	}
}
