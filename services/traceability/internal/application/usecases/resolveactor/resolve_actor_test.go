package resolveactor_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"rutadeorigen/traceability/internal/application/usecases/resolveactor"
	"rutadeorigen/traceability/internal/domain/actor"
	"rutadeorigen/traceability/internal/platform/exception"
)

var errInfra = errors.New("infrastructure failure")

type fakeActors struct {
	byUserID  map[string]*actor.Actor
	err       error
	calls     int
	lastQuery string
}

func (r *fakeActors) FindByID(_ context.Context, _ uuid.UUID) (*actor.Actor, error) {
	return actor.Default(), nil
}

func (r *fakeActors) FindByUserID(_ context.Context, userID string) (*actor.Actor, error) {
	r.calls++
	r.lastQuery = userID
	if r.err != nil {
		return actor.Default(), r.err
	}
	if a, ok := r.byUserID[userID]; ok {
		return a, nil
	}
	return actor.Default(), nil
}

func (r *fakeActors) Save(_ context.Context, _ *actor.Actor) error {
	return nil
}

func newActors() *fakeActors {
	return &fakeActors{byUserID: map[string]*actor.Actor{}}
}

func storedActor(userID string, status actor.Status) *actor.Actor {
	return actor.Restore(uuid.New(), userID, actor.TypeProducer, "Finca La Esperanza SAS", userID+"@example.com", status, time.Now().UTC())
}

func assertDefault(t *testing.T, got *actor.Actor) {
	t.Helper()
	if !got.IsDefault() {
		t.Error("expected the default actor on error")
	}
}

func assertCode(t *testing.T, got *actor.Actor, err error, code exception.Code) {
	t.Helper()
	if !exception.HasCode(err, code) {
		t.Errorf("got error %v, want code %s", err, code)
	}
	assertDefault(t, got)
}

func TestResolveActorReturnsActiveActor(t *testing.T) {
	actors := newActors()
	stored := storedActor("user_01", actor.StatusActive)
	actors.byUserID["user_01"] = stored

	got, err := resolveactor.New(actors).Execute(context.Background(), "user_01")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != stored {
		t.Error("expected the stored actor")
	}
}

func TestResolveActorTrimsUserID(t *testing.T) {
	actors := newActors()
	stored := storedActor("user_01", actor.StatusActive)
	actors.byUserID["user_01"] = stored

	got, err := resolveactor.New(actors).Execute(context.Background(), "  user_01  ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != stored {
		t.Error("expected the stored actor")
	}
	if actors.lastQuery != "user_01" {
		t.Errorf("got query %q, want %q", actors.lastQuery, "user_01")
	}
}

func TestResolveActorRequiresUserID(t *testing.T) {
	for name, userID := range map[string]string{"empty": "", "spaces": "   "} {
		t.Run(name, func(t *testing.T) {
			actors := newActors()

			got, err := resolveactor.New(actors).Execute(context.Background(), userID)

			assertCode(t, got, err, exception.CodeUnauthorized)
			if actors.calls != 0 {
				t.Error("expected no repository access")
			}
		})
	}
}

func TestResolveActorNotRegistered(t *testing.T) {
	actors := newActors()
	actors.byUserID["other_user"] = storedActor("other_user", actor.StatusActive)

	got, err := resolveactor.New(actors).Execute(context.Background(), "user_01")

	assertCode(t, got, err, exception.CodeActorNotFound)
}

func TestResolveActorInactive(t *testing.T) {
	actors := newActors()
	actors.byUserID["user_01"] = storedActor("user_01", actor.StatusInactive)

	got, err := resolveactor.New(actors).Execute(context.Background(), "user_01")

	assertCode(t, got, err, exception.CodeForbidden)
	if exception.HasCode(err, exception.CodeActorNotFound) {
		t.Error("an inactive actor must not be reported as not found")
	}
}

func TestResolveActorUnknownStatusIsRejected(t *testing.T) {
	actors := newActors()
	actors.byUserID["user_01"] = storedActor("user_01", actor.StatusUnknown)

	got, err := resolveactor.New(actors).Execute(context.Background(), "user_01")

	assertCode(t, got, err, exception.CodeForbidden)
}

func TestResolveActorRepositoryFails(t *testing.T) {
	actors := newActors()
	actors.err = errInfra

	got, err := resolveactor.New(actors).Execute(context.Background(), "user_01")

	e, ok := exception.From(err)
	if !ok || e.Kind() != exception.KindInternal {
		t.Errorf("expected an internal error, got %v", err)
	}
	if !errors.Is(err, errInfra) {
		t.Error("expected the original cause to be kept")
	}
	assertDefault(t, got)
}
