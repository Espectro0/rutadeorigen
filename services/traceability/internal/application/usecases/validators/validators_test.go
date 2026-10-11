package validators_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"rutadeorigen/traceability/internal/application/usecases"
	"rutadeorigen/traceability/internal/application/usecases/validators"
	"rutadeorigen/traceability/internal/domain/actor"
	"rutadeorigen/traceability/internal/domain/farm"
	"rutadeorigen/traceability/internal/platform/exception"
)

var errDatabase = errors.New("database is down")

type fakeActorRepository struct {
	actor *actor.Actor
	err   error
}

func (r fakeActorRepository) FindByID(_ context.Context, _ uuid.UUID) (*actor.Actor, error) {
	return r.actor, r.err
}

type fakeFarmRepository struct {
	farm *farm.Farm
	err  error
}

func (r fakeFarmRepository) FindByID(_ context.Context, _ uuid.UUID) (*farm.Farm, error) {
	return r.farm, r.err
}

func assertKind(t *testing.T, err error, want exception.Kind) {
	t.Helper()
	e, ok := exception.From(err)
	if !ok {
		t.Fatalf("expected an *Exception, got %v", err)
	}
	if e.Kind() != want {
		t.Errorf("Kind: got %v, want %v", e.Kind(), want)
	}
}

func TestActorActive(t *testing.T) {
	active, _ := actor.New("user_01", actor.TypeProducer, "Finca La Esperanza")
	inactive := actor.Restore(uuid.New(), "user_02", actor.TypeRoaster, "Tostadora Andina",
		actor.StatusInactive, time.Now())

	t.Run("returns the actor when it exists and is active", func(t *testing.T) {
		got, err := validators.ActorActive(context.Background(), fakeActorRepository{actor: active}, active.ID())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.ID() != active.ID() {
			t.Errorf("got actor %s, want %s", got.ID(), active.ID())
		}
	})

	t.Run("actor not found", func(t *testing.T) {
		got, err := validators.ActorActive(context.Background(), fakeActorRepository{actor: actor.Default()}, uuid.New())
		if !errors.Is(err, usecases.ErrActorNotFound) {
			t.Errorf("got %v, want %v", err, usecases.ErrActorNotFound)
		}
		if !got.IsDefault() {
			t.Error("expected default actor on error")
		}
	})

	t.Run("inactive actor counts as not found", func(t *testing.T) {
		got, err := validators.ActorActive(context.Background(), fakeActorRepository{actor: inactive}, inactive.ID())
		if !errors.Is(err, usecases.ErrActorNotFound) {
			t.Errorf("got %v, want %v", err, usecases.ErrActorNotFound)
		}
		if !got.IsDefault() {
			t.Error("expected default actor on error")
		}
	})

	t.Run("nil actor from repository counts as not found", func(t *testing.T) {
		_, err := validators.ActorActive(context.Background(), fakeActorRepository{actor: nil}, uuid.New())
		if !errors.Is(err, usecases.ErrActorNotFound) {
			t.Errorf("got %v, want %v", err, usecases.ErrActorNotFound)
		}
	})

	t.Run("repository failure is an internal error", func(t *testing.T) {
		got, err := validators.ActorActive(context.Background(), fakeActorRepository{err: errDatabase}, uuid.New())
		assertKind(t, err, exception.KindInternal)
		if !errors.Is(err, errDatabase) {
			t.Error("expected the original cause to be kept")
		}
		if !got.IsDefault() {
			t.Error("expected default actor on error")
		}
	})
}

func TestFarmExists(t *testing.T) {
	existing, _ := farm.New(uuid.New(), "La Esperanza", 1650)

	t.Run("returns the farm when it exists", func(t *testing.T) {
		got, err := validators.FarmExists(context.Background(), fakeFarmRepository{farm: existing}, existing.ID())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.ID() != existing.ID() {
			t.Errorf("got farm %s, want %s", got.ID(), existing.ID())
		}
	})

	t.Run("farm not found", func(t *testing.T) {
		got, err := validators.FarmExists(context.Background(), fakeFarmRepository{farm: farm.Default()}, uuid.New())
		if !errors.Is(err, usecases.ErrFarmNotFound) {
			t.Errorf("got %v, want %v", err, usecases.ErrFarmNotFound)
		}
		if !got.IsDefault() {
			t.Error("expected default farm on error")
		}
	})

	t.Run("nil farm from repository counts as not found", func(t *testing.T) {
		_, err := validators.FarmExists(context.Background(), fakeFarmRepository{farm: nil}, uuid.New())
		if !errors.Is(err, usecases.ErrFarmNotFound) {
			t.Errorf("got %v, want %v", err, usecases.ErrFarmNotFound)
		}
	})

	t.Run("repository failure is an internal error", func(t *testing.T) {
		got, err := validators.FarmExists(context.Background(), fakeFarmRepository{err: errDatabase}, uuid.New())
		assertKind(t, err, exception.KindInternal)
		if !errors.Is(err, errDatabase) {
			t.Error("expected the original cause to be kept")
		}
		if !got.IsDefault() {
			t.Error("expected default farm on error")
		}
	})
}
