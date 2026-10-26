package validators_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"rutadeorigen/traceability/internal/application/usecases/validators"
	"rutadeorigen/traceability/internal/domain/actor"
	"rutadeorigen/traceability/internal/domain/farm"
	"rutadeorigen/traceability/internal/domain/lot"
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

func (r fakeFarmRepository) Save(_ context.Context, _ *farm.Farm) error {
	return nil
}

func (r fakeFarmRepository) ListByProducer(_ context.Context, _ uuid.UUID) ([]*farm.Farm, error) {
	return []*farm.Farm{}, nil
}

type fakeLotRepository struct {
	lot *lot.Lot
	err error
}

func (r fakeLotRepository) ExistsByCode(_ context.Context, _ uuid.UUID, _ string) (bool, error) {
	return false, nil
}

func (r fakeLotRepository) UpdateStage(_ context.Context, _ *lot.Lot, _ int) error {
	return nil
}

func (r fakeLotRepository) Save(_ context.Context, _ *lot.Lot) error {
	return nil
}

func (r fakeLotRepository) FindByID(_ context.Context, _ uuid.UUID) (*lot.Lot, error) {
	return r.lot, r.err
}

func (r fakeLotRepository) List(_ context.Context, _ lot.Filter, _ lot.Cursor, _ int) ([]*lot.Lot, error) {
	return nil, nil
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
	active, _ := actor.New("user_01", actor.TypeProducer, "Finca La Esperanza", "user_01@example.com")
	inactive := actor.Restore(uuid.New(), "user_02", actor.TypeRoaster, "Tostadora Andina", "user_02@example.com",
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
		if !errors.Is(err, exception.New(exception.CodeActorNotFound)) {
			t.Errorf("got %v, want %v", err, exception.New(exception.CodeActorNotFound))
		}
		if !got.IsDefault() {
			t.Error("expected default actor on error")
		}
	})

	t.Run("inactive actor counts as not found", func(t *testing.T) {
		got, err := validators.ActorActive(context.Background(), fakeActorRepository{actor: inactive}, inactive.ID())
		if !errors.Is(err, exception.New(exception.CodeActorNotFound)) {
			t.Errorf("got %v, want %v", err, exception.New(exception.CodeActorNotFound))
		}
		if !got.IsDefault() {
			t.Error("expected default actor on error")
		}
	})

	t.Run("nil actor from repository counts as not found", func(t *testing.T) {
		_, err := validators.ActorActive(context.Background(), fakeActorRepository{actor: nil}, uuid.New())
		if !errors.Is(err, exception.New(exception.CodeActorNotFound)) {
			t.Errorf("got %v, want %v", err, exception.New(exception.CodeActorNotFound))
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
		if !errors.Is(err, exception.New(exception.CodeFarmNotFound)) {
			t.Errorf("got %v, want %v", err, exception.New(exception.CodeFarmNotFound))
		}
		if !got.IsDefault() {
			t.Error("expected default farm on error")
		}
	})

	t.Run("nil farm from repository counts as not found", func(t *testing.T) {
		_, err := validators.FarmExists(context.Background(), fakeFarmRepository{farm: nil}, uuid.New())
		if !errors.Is(err, exception.New(exception.CodeFarmNotFound)) {
			t.Errorf("got %v, want %v", err, exception.New(exception.CodeFarmNotFound))
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

func newValidatorsLot(t *testing.T) *lot.Lot {
	t.Helper()
	l, err := lot.New(uuid.New(), uuid.New(), "LT-001", "Castillo", lot.ProcessWashed, time.Now().AddDate(0, -1, 0))
	if err != nil {
		t.Fatal(err)
	}
	return l
}

func TestLotExists(t *testing.T) {
	existing := newValidatorsLot(t)

	t.Run("returns the lot when it exists", func(t *testing.T) {
		got, err := validators.LotExists(context.Background(), fakeLotRepository{lot: existing}, existing.ID())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.ID() != existing.ID() {
			t.Errorf("got lot %s, want %s", got.ID(), existing.ID())
		}
	})

	t.Run("lot not found", func(t *testing.T) {
		got, err := validators.LotExists(context.Background(), fakeLotRepository{lot: lot.Default()}, uuid.New())
		if !errors.Is(err, exception.New(exception.CodeLotNotFound)) {
			t.Errorf("got %v, want %v", err, exception.New(exception.CodeLotNotFound))
		}
		if !got.IsDefault() {
			t.Error("expected default lot on error")
		}
	})

	t.Run("nil lot from repository counts as not found", func(t *testing.T) {
		_, err := validators.LotExists(context.Background(), fakeLotRepository{lot: nil}, uuid.New())
		if !errors.Is(err, exception.New(exception.CodeLotNotFound)) {
			t.Errorf("got %v, want %v", err, exception.New(exception.CodeLotNotFound))
		}
	})

	t.Run("repository failure is an internal error", func(t *testing.T) {
		got, err := validators.LotExists(context.Background(), fakeLotRepository{err: errDatabase}, uuid.New())
		assertKind(t, err, exception.KindInternal)
		if !errors.Is(err, errDatabase) {
			t.Error("expected the original cause to be kept")
		}
		if !got.IsDefault() {
			t.Error("expected default lot on error")
		}
	})
}

func TestLotVersion(t *testing.T) {
	l := newValidatorsLot(t)

	t.Run("the version matches", func(t *testing.T) {
		if err := validators.LotVersion(l, 1); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("the version matches after the lot advances", func(t *testing.T) {
		advanced := newValidatorsLot(t)
		if err := advanced.AdvanceTo(lot.StageFarm); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := validators.LotVersion(advanced, 2); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if err := validators.LotVersion(advanced, 1); !exception.HasCode(err, exception.CodeLotVersionConflict) {
			t.Errorf("got %v, want code %q", err, exception.CodeLotVersionConflict)
		}
	})

	for _, expected := range []int{-1, 0, 2, 99} {
		t.Run("a different version conflicts", func(t *testing.T) {
			err := validators.LotVersion(l, expected)
			if !exception.HasCode(err, exception.CodeLotVersionConflict) {
				t.Fatalf("expected %d: got %v, want code %q", expected, err, exception.CodeLotVersionConflict)
			}
			assertKind(t, err, exception.KindConflict)
		})
	}

	t.Run("a default lot is not found even with a zero version", func(t *testing.T) {
		if err := validators.LotVersion(lot.Default(), 0); !exception.HasCode(err, exception.CodeLotNotFound) {
			t.Errorf("got %v, want code %q", err, exception.CodeLotNotFound)
		}
	})

	t.Run("a nil lot is not found", func(t *testing.T) {
		if err := validators.LotVersion(nil, 1); !exception.HasCode(err, exception.CodeLotNotFound) {
			t.Errorf("got %v, want code %q", err, exception.CodeLotNotFound)
		}
	})
}
