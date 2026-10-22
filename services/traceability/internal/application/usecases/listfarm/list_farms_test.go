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
	byProducer map[uuid.UUID][]*farm.Farm
	listErr    error
	nilList    bool
	askedFor   uuid.UUID
}

func (r *fakeFarms) FindByID(_ context.Context, _ uuid.UUID) (*farm.Farm, error) {
	return farm.Default(), nil
}

func (r *fakeFarms) Save(_ context.Context, _ *farm.Farm) error {
	return nil
}

func (r *fakeFarms) ListByProducer(_ context.Context, producerID uuid.UUID) ([]*farm.Farm, error) {
	r.askedFor = producerID
	if r.listErr != nil {
		return nil, r.listErr
	}
	if r.nilList {
		return nil, nil
	}
	return r.byProducer[producerID], nil
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
	own        []*farm.Farm
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

	producer, err := actor.New("user_01", actor.TypeProducer, "Finca La Esperanza SAS")
	if err != nil {
		t.Fatal(err)
	}
	other := uuid.New()
	own := []*farm.Farm{newFarm(t, producer.ID(), "La Esperanza"), newFarm(t, producer.ID(), "El Mirador")}

	return &scenario{
		producer: producer,
		own:      own,
		actors:   fakeActors{byID: map[uuid.UUID]*actor.Actor{producer.ID(): producer}},
		farms: &fakeFarms{byProducer: map[uuid.UUID][]*farm.Farm{
			producer.ID(): own,
			other:         {newFarm(t, other, "La Otra Finca")},
		}},
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

	t.Run("returns only the farms of the actor", func(t *testing.T) {
		if len(got) != 2 || got[0].ID() != s.own[0].ID() || got[1].ID() != s.own[1].ID() {
			t.Errorf("got %d farms, want the 2 farms of the actor", len(got))
		}
		for _, f := range got {
			if f.ProducerID() != s.producer.ID() {
				t.Error("a farm of another producer was returned")
			}
		}
	})

	t.Run("asks the repository for the actor farms", func(t *testing.T) {
		if s.farms.askedFor != s.producer.ID() {
			t.Errorf("got producer %s, want %s", s.farms.askedFor, s.producer.ID())
		}
	})

	t.Run("asks ABAC to read farms owned by the actor", func(t *testing.T) {
		a := s.authorizer
		if a.action != ports.ActionReadFarm {
			t.Errorf("action: got %v, want %v", a.action, ports.ActionReadFarm)
		}
		if a.subject.ActorID != s.producer.ID() || a.subject.Role != actor.TypeProducer {
			t.Error("unexpected subject")
		}
		if a.resource.Type != ports.ResourceFarm || a.resource.OwnerID != s.producer.ID() {
			t.Error("unexpected resource")
		}
	})
}

func TestListFarmsEmpty(t *testing.T) {
	t.Run("an actor without farms gets an empty list", func(t *testing.T) {
		s := newScenario(t)
		roaster, err := actor.New("user_02", actor.TypeRoaster, "Tostadora del Oriente")
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
			"Finca La Esperanza SAS", actor.StatusInactive, time.Now())

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
		if s.farms.askedFor != uuid.Nil {
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
