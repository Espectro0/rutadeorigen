package createfarm_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"rutadeorigen/lib/requestcontext"
	"rutadeorigen/traceability/internal/application/output/ports"
	"rutadeorigen/traceability/internal/application/usecases/createfarm"
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
	saveErr error
	saved   []*farm.Farm
}

func (r *fakeFarms) FindByID(_ context.Context, _ uuid.UUID) (*farm.Farm, error) {
	return farm.Default(), nil
}

func (r *fakeFarms) Save(_ context.Context, f *farm.Farm) error {
	if r.saveErr != nil {
		return r.saveErr
	}
	r.saved = append(r.saved, f)
	return nil
}

func (r *fakeFarms) ListByProducer(_ context.Context, _ uuid.UUID) ([]*farm.Farm, error) {
	return []*farm.Farm{}, nil
}

type fakeAuthorizer struct {
	allowed  bool
	err      error
	calls    int
	subject  ports.Subject
	action   ports.Action
	resource ports.Resource
}

func (a *fakeAuthorizer) IsAllowed(_ context.Context, s ports.Subject, act ports.Action, r ports.Resource) (bool, error) {
	a.calls++
	a.subject, a.action, a.resource = s, act, r
	return a.allowed, a.err
}

type scenario struct {
	producer   *actor.Actor
	actors     fakeActors
	farms      *fakeFarms
	authorizer *fakeAuthorizer
}

func newScenario(t *testing.T) *scenario {
	t.Helper()

	producer, err := actor.New("user_01", actor.TypeProducer, "Finca La Esperanza SAS", "user_01@example.com")
	if err != nil {
		t.Fatal(err)
	}

	return &scenario{
		producer:   producer,
		actors:     fakeActors{byID: map[uuid.UUID]*actor.Actor{producer.ID(): producer}},
		farms:      &fakeFarms{},
		authorizer: &fakeAuthorizer{allowed: true},
	}
}

func (s *scenario) useCase() *createfarm.CreateFarm {
	return createfarm.New(s.actors, s.farms, s.authorizer)
}

func (s *scenario) ctx() context.Context {
	return requestcontext.WithActorID(context.Background(), s.producer.ID())
}

func execute(ctx context.Context, uc createfarm.UseCase, draft *farm.Farm) (*farm.Farm, error) {
	err := uc.Execute(ctx, draft)
	return draft, err
}

func assertFailed(t *testing.T, got *farm.Farm, err, want error) {
	t.Helper()
	if !errors.Is(err, want) {
		t.Errorf("got error %v, want %v", err, want)
	}
	if !got.IsDefault() {
		t.Error("expected the default farm on error")
	}
}

func assertInternal(t *testing.T, got *farm.Farm, err error) {
	t.Helper()
	e, ok := exception.From(err)
	if !ok || e.Kind() != exception.KindInternal {
		t.Errorf("expected an internal error, got %v", err)
	}
	if !errors.Is(err, errInfra) {
		t.Error("expected the original cause to be kept")
	}
	if !got.IsDefault() {
		t.Error("expected the default farm on error")
	}
}

func TestCreateFarmSuccess(t *testing.T) {
	s := newScenario(t)

	got, err := execute(s.ctx(), s.useCase(), farm.Draft("  La Esperanza  ", 1650))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Run("returns the created farm", func(t *testing.T) {
		if got.IsDefault() {
			t.Fatal("expected a real farm")
		}
		if got.Name() != "La Esperanza" || got.Altitude() != 1650 {
			t.Errorf("unexpected farm data: %q %d", got.Name(), got.Altitude())
		}
		if got.CreatedAt().IsZero() {
			t.Error("expected a creation date")
		}
	})

	t.Run("the producer is the actor of the request", func(t *testing.T) {
		if got.ProducerID() != s.producer.ID() {
			t.Errorf("got producer %s, want %s", got.ProducerID(), s.producer.ID())
		}
	})

	t.Run("saves the farm", func(t *testing.T) {
		if len(s.farms.saved) != 1 || s.farms.saved[0].ID() != got.ID() {
			t.Error("expected the created farm to be saved once")
		}
	})

	t.Run("asks ABAC to create a farm owned by the actor", func(t *testing.T) {
		a := s.authorizer
		if a.action != ports.ActionCreateFarm {
			t.Errorf("action: got %v, want %v", a.action, ports.ActionCreateFarm)
		}
		if a.subject.ActorID != s.producer.ID() || a.subject.Role != actor.TypeProducer {
			t.Error("unexpected subject")
		}
		if a.resource.Type != ports.ResourceFarm || a.resource.OwnerID != s.producer.ID() {
			t.Error("unexpected resource")
		}
	})
}

func TestCreateFarmActorErrors(t *testing.T) {
	t.Run("no actor in the context", func(t *testing.T) {
		s := newScenario(t)
		got, err := execute(context.Background(), s.useCase(), farm.Draft("La Esperanza", 1650))
		assertFailed(t, got, err, exception.New(exception.CodeActorNotFound))
	})

	t.Run("inactive actor", func(t *testing.T) {
		s := newScenario(t)
		s.actors.byID[s.producer.ID()] = actor.Restore(s.producer.ID(), "user_01", actor.TypeProducer,
			"Finca La Esperanza SAS", "user_01@example.com", actor.StatusInactive, time.Now())

		got, err := execute(s.ctx(), s.useCase(), farm.Draft("La Esperanza", 1650))
		assertFailed(t, got, err, exception.New(exception.CodeActorNotFound))
	})

	t.Run("actor repository fails", func(t *testing.T) {
		s := newScenario(t)
		s.actors.err = errInfra
		got, err := execute(s.ctx(), s.useCase(), farm.Draft("La Esperanza", 1650))
		assertInternal(t, got, err)
	})

	t.Run("the actor is not a producer: nothing saved and ABAC not asked", func(t *testing.T) {
		s := newScenario(t)
		roaster, err := actor.New("user_02", actor.TypeRoaster, "Tostadora del Oriente", "user_02@example.com")
		if err != nil {
			t.Fatal(err)
		}
		s.actors.byID[roaster.ID()] = roaster
		ctx := requestcontext.WithActorID(context.Background(), roaster.ID())

		got, err := execute(ctx, s.useCase(), farm.Draft("La Esperanza", 1650))
		assertFailed(t, got, err, exception.New(exception.CodeFarmInvalidProducer))
		if len(s.farms.saved) != 0 {
			t.Error("nothing must be saved when the actor is not a producer")
		}
		if s.authorizer.calls != 0 {
			t.Error("ABAC must not be asked when the actor is not a producer")
		}
	})
}

func TestCreateFarmAuthorization(t *testing.T) {
	t.Run("ABAC denies: forbidden and nothing saved", func(t *testing.T) {
		s := newScenario(t)
		s.authorizer.allowed = false

		got, err := execute(s.ctx(), s.useCase(), farm.Draft("La Esperanza", 1650))
		assertFailed(t, got, err, exception.New(exception.CodeForbidden))

		e, _ := exception.From(err)
		if e.Kind() != exception.KindForbidden {
			t.Errorf("Kind: got %v, want %v", e.Kind(), exception.KindForbidden)
		}
		if len(s.farms.saved) != 0 {
			t.Error("nothing must be saved when access is denied")
		}
	})

	t.Run("ABAC fails", func(t *testing.T) {
		s := newScenario(t)
		s.authorizer.err = errInfra
		got, err := execute(s.ctx(), s.useCase(), farm.Draft("La Esperanza", 1650))
		assertInternal(t, got, err)
	})
}

func TestCreateFarmDomainErrors(t *testing.T) {
	tests := []struct {
		name  string
		draft *farm.Farm
		want  error
	}{
		{"name too short", farm.Draft("LE", 1650), exception.New(exception.CodeFarmInvalidName)},
		{"blank name", farm.Draft("   ", 1650), exception.New(exception.CodeFarmInvalidName)},
		{"negative altitude", farm.Draft("La Esperanza", -1), exception.New(exception.CodeFarmInvalidAltitude)},
		{"altitude above the limit", farm.Draft("La Esperanza", 3001), exception.New(exception.CodeFarmInvalidAltitude)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newScenario(t)

			got, err := execute(s.ctx(), s.useCase(), tt.draft)
			assertFailed(t, got, err, tt.want)
			if len(s.farms.saved) != 0 {
				t.Error("nothing must be saved when the domain rejects the farm")
			}
		})
	}
}

func TestCreateFarmPersistenceErrors(t *testing.T) {
	t.Run("saving fails", func(t *testing.T) {
		s := newScenario(t)
		s.farms.saveErr = errInfra
		got, err := execute(s.ctx(), s.useCase(), farm.Draft("La Esperanza", 1650))
		assertInternal(t, got, err)
	})

	t.Run("a domain error from the repository keeps its code", func(t *testing.T) {
		s := newScenario(t)
		s.farms.saveErr = exception.New(exception.CodeFarmInvalidName)
		got, err := execute(s.ctx(), s.useCase(), farm.Draft("La Esperanza", 1650))
		assertFailed(t, got, err, exception.New(exception.CodeFarmInvalidName))
	})
}

func TestCreateFarmNilDraft(t *testing.T) {
	s := newScenario(t)

	got, err := execute(s.ctx(), s.useCase(), nil)
	assertFailed(t, got, err, exception.New(exception.CodeFarmInvalidName))
}
