package registeractor_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"rutadeorigen/lib/requestcontext"
	"rutadeorigen/traceability/internal/application/input/adapters/registeractor"
	"rutadeorigen/traceability/internal/application/input/dto/request"
	"rutadeorigen/traceability/internal/application/input/dto/response"
	registeractoruc "rutadeorigen/traceability/internal/application/usecases/registeractor"
	"rutadeorigen/traceability/internal/domain/actor"
	"rutadeorigen/traceability/internal/platform/exception"
)

type ctxKey struct{}

type fakeUseCase struct {
	result registeractoruc.Result
	err    error

	calls    int
	received registeractoruc.Command
	ctx      context.Context
}

func (f *fakeUseCase) Execute(ctx context.Context, cmd registeractoruc.Command) (registeractoruc.Result, error) {
	f.calls++
	f.received = cmd
	f.ctx = ctx
	if f.err != nil {
		return registeractoruc.Result{}, f.err
	}
	return f.result, nil
}

func restoredActor() *actor.Actor {
	return actor.Restore(uuid.New(), "user_01", actor.TypeProducer, "Finca La Esperanza SAS", "producer@example.com", actor.StatusActive, time.Date(2026, time.September, 20, 14, 30, 0, 0, time.UTC))
}

func sessionContext() context.Context {
	ctx := requestcontext.WithUserID(context.Background(), "user_01")
	return requestcontext.WithEmail(ctx, "producer@example.com")
}

func validInput() request.RegisterActor {
	return request.RegisterActor{Type: "producer", Name: "Finca La Esperanza SAS"}
}

func TestAdapterExecute(t *testing.T) {
	t.Run("builds the command from the session and the body", func(t *testing.T) {
		uc := &fakeUseCase{result: registeractoruc.Result{Actor: restoredActor(), Created: true}}

		if _, err := registeractor.New(uc).Execute(sessionContext(), validInput()); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		want := registeractoruc.Command{
			UserID: "user_01",
			Email:  "producer@example.com",
			Type:   actor.TypeProducer,
			Name:   "Finca La Esperanza SAS",
		}
		if uc.calls != 1 || uc.received != want {
			t.Errorf("got %d calls with %+v, want 1 call with %+v", uc.calls, uc.received, want)
		}
	})

	t.Run("returns the mapped actor and the created flag", func(t *testing.T) {
		stored := restoredActor()
		uc := &fakeUseCase{result: registeractoruc.Result{Actor: stored, Created: true}}

		got, err := registeractor.New(uc).Execute(sessionContext(), validInput())

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !got.Created {
			t.Error("expected Created to be true")
		}
		want := response.Actor{
			ID:        stored.ID().String(),
			Type:      "producer",
			Name:      "Finca La Esperanza SAS",
			Email:     "producer@example.com",
			Status:    "active",
			CreatedAt: "2026-09-20T14:30:00Z",
		}
		if got.Actor != want {
			t.Errorf("got %+v, want %+v", got.Actor, want)
		}
	})

	t.Run("an existing actor is not reported as created", func(t *testing.T) {
		uc := &fakeUseCase{result: registeractoruc.Result{Actor: restoredActor()}}

		got, err := registeractor.New(uc).Execute(sessionContext(), validInput())

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Created {
			t.Error("expected Created to be false")
		}
	})

	t.Run("passes the same context to the use case", func(t *testing.T) {
		uc := &fakeUseCase{result: registeractoruc.Result{Actor: restoredActor()}}
		ctx := context.WithValue(sessionContext(), ctxKey{}, "marker")

		if _, err := registeractor.New(uc).Execute(ctx, validInput()); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if uc.ctx.Value(ctxKey{}) != "marker" {
			t.Error("the adapter must pass the same context to the use case")
		}
	})

	t.Run("without a session passes empty identity", func(t *testing.T) {
		uc := &fakeUseCase{err: exception.New(exception.CodeUnauthorized)}

		_, err := registeractor.New(uc).Execute(context.Background(), validInput())

		if uc.received.UserID != "" || uc.received.Email != "" {
			t.Errorf("got identity %q / %q, want empty", uc.received.UserID, uc.received.Email)
		}
		if !exception.HasCode(err, exception.CodeUnauthorized) {
			t.Errorf("got error %v, want unauthorized", err)
		}
	})

	t.Run("an unknown type reaches the use case as is", func(t *testing.T) {
		uc := &fakeUseCase{err: exception.New(exception.CodeActorInvalidType)}

		_, err := registeractor.New(uc).Execute(sessionContext(), request.RegisterActor{Type: "farmer", Name: "Finca"})

		if uc.received.Type != actor.Type("farmer") {
			t.Errorf("got type %q, want farmer", uc.received.Type)
		}
		if !exception.HasCode(err, exception.CodeActorInvalidType) {
			t.Errorf("got error %v, want invalid type", err)
		}
	})

	useCaseErrors := []struct {
		name string
		err  error
	}{
		{"invalid name", exception.New(exception.CodeActorInvalidName)},
		{"invalid email", exception.New(exception.CodeActorInvalidEmail)},
		{"unauthorized", exception.New(exception.CodeUnauthorized)},
		{"internal", exception.New(exception.CodeInternal).WithTechnical("boom")},
		{"plain error", errors.New("boom")},
	}

	for _, tt := range useCaseErrors {
		t.Run("propagates "+tt.name, func(t *testing.T) {
			uc := &fakeUseCase{err: tt.err}

			got, err := registeractor.New(uc).Execute(sessionContext(), validInput())

			if !errors.Is(err, tt.err) {
				t.Errorf("got error %v, want %v", err, tt.err)
			}
			if got != (response.ActorRegistration{}) {
				t.Errorf("got %+v, want empty registration", got)
			}
		})
	}
}
