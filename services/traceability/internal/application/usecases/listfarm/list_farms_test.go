package listfarm_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"rutadeorigen/lib/requestcontext"
	"rutadeorigen/traceability/internal/application/output/ports"
	"rutadeorigen/traceability/internal/application/usecases/listfarm"
	"rutadeorigen/traceability/internal/domain/actor"
	"rutadeorigen/traceability/internal/domain/farm"
	"rutadeorigen/traceability/internal/platform/exception"
)

var errInfra = errors.New("infrastructure failure")

type fakeActors struct {
	byID map[uuid.UUID]*actor.Actor
	err  error
}

func (r fakeActors) FindByID(_ context.Context, id uuid.UUID) (*actor.Actor, error) {
	if r.err != nil {
		return actor.Default(), r.err
	}
	if a, ok := r.byID[id]; ok {
		return a, nil
	}
	return actor.Default(), nil
}

type fakeFarms struct {
	all     []*farm.Farm
	listErr error
	nilList bool
	calls   int
}

func (r *fakeFarms) FindByID(_ context.Context, _ uuid.UUID) (*farm.Farm, error) {
	return farm.Default(), nil
}

func (r *fakeFarms) Save(_ context.Context, _ *farm.Farm) error {
	return nil
}

func (r *fakeFarms) Delete(_ context.Context, _ uuid.UUID) error { return nil }

func (r *fakeFarms) List(_ context.Context) ([]*farm.Farm, error) {
	r.calls++
	if r.listErr != nil {
		return nil, r.listErr
	}
	if r.nilList {
		return nil, nil
	}
	return r.all, nil
}

type fakeAuthorizer struct {
	allowed  bool
	err      error
	subject  ports.Subject
	action   ports.Action
	resource ports.Resource
}

func (a *fakeAuthorizer) IsAllowed(_ context.Context, s ports.Subject, act ports.Action, r ports.Resource) (bool, error) {
	a.subject, a.action, a.resource = s, act, r
	return a.allowed, a.err
}

type scenario struct {
	producer   *actor.Actor
	farmsOfAll []*farm.Farm
	actors     fakeActors
	farms      *fakeFarms
	authorizer *fakeAuthorizer
}

func newFarm(t *testing.T, producerID uuid.UUID, name string) *farm.Farm {
	t.Helper()
	f, err := farm.New(producerID, name, 1650)
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func newScenario(t *testing.T) *scenario {
	t.Helper()

	producer, err := actor.New("user_01", actor.TypeProducer, "Finca La Esperanza SAS", "user_01@example.com")
	if err != nil {
		t.Fatal(err)
	}
	all := []*farm.Farm{
		newFarm(t, producer.ID(), "La Esperanza"),
		newFarm(t, producer.ID(), "El Mirador"),
		newFarm(t, uuid.New(), "La Otra Finca"),
	}

	return &scenario{
		producer:   producer,
		farmsOfAll: all,
		actors:     fakeActors{byID: map[uuid.UUID]*actor.Actor{producer.ID(): producer}},
		farms:      &fakeFarms{all: all},
		authorizer: &fakeAuthorizer{allowed: true},
	}
}

func (s *scenario) useCase() *listfarm.ListFarms {
	return listfarm.New(s.actors, s.farms, s.authorizer)
}

func (s *scenario) ctx() context.Context {
	return requestcontext.WithActorID(context.Background(), s.producer.ID())
}

func assertFailed(t *testing.T, got []*farm.Farm, err, want error) {
	t.Helper()
	if !errors.Is(err, want) {
		t.Errorf("got error %v, want %v", err, want)
	}
	if got == nil || len(got) != 0 {
		t.Error("expected an empty, non-nil list on error")
	}
}

func assertInternal(t *testing.T, got []*farm.Farm, err error) {
	t.Helper()
	e, ok := exception.From(err)
	if !ok || e.Kind() != exception.KindInternal {
		t.Errorf("expected an internal error, got %v", err)
	}
	if !errors.Is(err, errInfra) {
		t.Error("expected the original cause to be kept")
	}
	if got == nil || len(got) != 0 {
		t.Error("expected an empty, non-nil list on error")
	}
}

func TestListFarmsSuccess(t *testing.T) {
	s := newScenario(t)

	got, err := s.useCase().Execute(s.ctx(), listfarm.Query{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Run("returns the farms of every producer", func(t *testing.T) {
		if len(got) != 3 {
			t.Fatalf("got %d farms, want 3", len(got))
		}
		for i, f := range got {
			if f.ID() != s.farmsOfAll[i].ID() {
				t.Errorf("farm %d: got %s, want %s", i, f.ID(), s.farmsOfAll[i].ID())
			}
		}
	})

	t.Run("asks the repository once", func(t *testing.T) {
		if s.farms.calls != 1 {
			t.Errorf("got %d repository calls, want 1", s.farms.calls)
		}
	})

	t.Run("asks ABAC to read farms without an owner", func(t *testing.T) {
		a := s.authorizer
		if a.action != ports.ActionReadFarm {
			t.Errorf("action: got %v, want %v", a.action, ports.ActionReadFarm)
		}
		if a.subject.ActorID != s.producer.ID() || a.subject.Role != actor.TypeProducer {
			t.Error("unexpected subject")
		}
		if a.resource != (ports.Resource{Type: ports.ResourceFarm}) {
			t.Error("unexpected resource")
		}
	})
}

func TestListFarmsEmpty(t *testing.T) {
	t.Run("without farms the list is empty", func(t *testing.T) {
		s := newScenario(t)
		s.farms.all = nil
		s.farms.nilList = false
		roaster, err := actor.New("user_02", actor.TypeRoaster, "Tostadora del Oriente", "user_02@example.com")
		if err != nil {
			t.Fatal(err)
		}
		s.actors.byID[roaster.ID()] = roaster
		ctx := requestcontext.WithActorID(context.Background(), roaster.ID())

		got, err := s.useCase().Execute(ctx, listfarm.Query{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got == nil || len(got) != 0 {
			t.Error("expected an empty, non-nil list")
		}
	})

	t.Run("a nil list from the repository becomes an empty list", func(t *testing.T) {
		s := newScenario(t)
		s.farms.nilList = true

		got, err := s.useCase().Execute(s.ctx(), listfarm.Query{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got == nil || len(got) != 0 {
			t.Error("expected an empty, non-nil list")
		}
	})
}

func TestListFarmsActorErrors(t *testing.T) {
	t.Run("no actor in the context", func(t *testing.T) {
		s := newScenario(t)
		got, err := s.useCase().Execute(context.Background(), listfarm.Query{})
		assertFailed(t, got, err, exception.New(exception.CodeActorNotFound))
	})

	t.Run("inactive actor", func(t *testing.T) {
		s := newScenario(t)
		s.actors.byID[s.producer.ID()] = actor.Restore(s.producer.ID(), "user_01", actor.TypeProducer,
			"Finca La Esperanza SAS", "user_01@example.com", actor.StatusInactive, time.Now())

		got, err := s.useCase().Execute(s.ctx(), listfarm.Query{})
		assertFailed(t, got, err, exception.New(exception.CodeActorNotFound))
	})

	t.Run("actor repository fails", func(t *testing.T) {
		s := newScenario(t)
		s.actors.err = errInfra
		got, err := s.useCase().Execute(s.ctx(), listfarm.Query{})
		assertInternal(t, got, err)
	})
}

func TestListFarmsAuthorization(t *testing.T) {
	t.Run("ABAC denies: forbidden and the repository is not asked", func(t *testing.T) {
		s := newScenario(t)
		s.authorizer.allowed = false

		got, err := s.useCase().Execute(s.ctx(), listfarm.Query{})
		assertFailed(t, got, err, exception.New(exception.CodeForbidden))
		if s.farms.calls != 0 {
			t.Error("the repository must not be asked when access is denied")
		}
	})

	t.Run("ABAC fails", func(t *testing.T) {
		s := newScenario(t)
		s.authorizer.err = errInfra
		got, err := s.useCase().Execute(s.ctx(), listfarm.Query{})
		assertInternal(t, got, err)
	})
}

func TestListFarmsRepositoryErrors(t *testing.T) {
	s := newScenario(t)
	s.farms.listErr = errInfra

	got, err := s.useCase().Execute(s.ctx(), listfarm.Query{})
	assertInternal(t, got, err)
}

func (r fakeActors) ListByType(_ context.Context, _ actor.Type) ([]*actor.Actor, error) {
	return nil, nil
}

func (r fakeActors) FindByUserID(_ context.Context, _ string) (*actor.Actor, error) {
	return actor.Default(), nil
}

func (r fakeActors) Save(_ context.Context, _ *actor.Actor) error {
	return nil
}
