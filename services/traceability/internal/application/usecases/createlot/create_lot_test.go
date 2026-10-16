package createlot_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"rutadeorigen/lib/requestcontext"
	"rutadeorigen/traceability/internal/application/output/ports"
	"rutadeorigen/traceability/internal/application/usecases/createlot"
	"rutadeorigen/traceability/internal/domain/actor"
	"rutadeorigen/traceability/internal/domain/farm"
	"rutadeorigen/traceability/internal/domain/lot"
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
	byID map[uuid.UUID]*farm.Farm
	err  error
}

func (r fakeFarms) FindByID(_ context.Context, id uuid.UUID) (*farm.Farm, error) {
	if r.err != nil {
		return farm.Default(), r.err
	}
	if f, ok := r.byID[id]; ok {
		return f, nil
	}
	return farm.Default(), nil
}

type fakeLots struct {
	existingCode string
	existsErr    error
	saveErr      error
	checkedCode  string
	saved        []*lot.Lot
}

func (r *fakeLots) ExistsByCode(_ context.Context, _ uuid.UUID, code string) (bool, error) {
	r.checkedCode = code
	if r.existsErr != nil {
		return false, r.existsErr
	}
	return code == r.existingCode, nil
}

func (r *fakeLots) Save(_ context.Context, l *lot.Lot) error {
	if r.saveErr != nil {
		return r.saveErr
	}
	r.saved = append(r.saved, l)
	return nil
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

type fakePublisher struct {
	err    error
	events []ports.Event
}

func (p *fakePublisher) Publish(_ context.Context, e ports.Event) error {
	if p.err != nil {
		return p.err
	}
	p.events = append(p.events, e)
	return nil
}

type scenario struct {
	producer   *actor.Actor
	farm       *farm.Farm
	actors     fakeActors
	farms      fakeFarms
	lots       *fakeLots
	authorizer *fakeAuthorizer
	publisher  *fakePublisher
}

func newScenario(t *testing.T) *scenario {
	t.Helper()

	producer, err := actor.New("user_01", actor.TypeProducer, "Finca La Esperanza SAS")
	if err != nil {
		t.Fatal(err)
	}
	f, err := farm.New(producer.ID(), "La Esperanza", 1650)
	if err != nil {
		t.Fatal(err)
	}

	return &scenario{
		producer:   producer,
		farm:       f,
		actors:     fakeActors{byID: map[uuid.UUID]*actor.Actor{producer.ID(): producer}},
		farms:      fakeFarms{byID: map[uuid.UUID]*farm.Farm{f.ID(): f}},
		lots:       &fakeLots{},
		authorizer: &fakeAuthorizer{allowed: true},
		publisher:  &fakePublisher{},
	}
}

func (s *scenario) useCase() *createlot.CreateLot {
	return createlot.New(s.actors, s.farms, s.lots, s.authorizer, s.publisher)
}

func (s *scenario) ctx() context.Context {
	ctx := requestcontext.WithActorID(context.Background(), s.producer.ID())
	return requestcontext.WithCorrelationID(ctx, "req-123")
}

func (s *scenario) draft() *lot.Lot {
	return lot.Draft(s.farm.ID(), "  LT-001  ", "Castillo", lot.ProcessWashed, time.Now().AddDate(0, -1, 0))
}

// execute runs the use case and returns the draft it filled in, which is the
// created lot on success and still the default lot on failure.
func execute(uc createlot.UseCase, ctx context.Context, draft *lot.Lot) (*lot.Lot, error) {
	err := uc.Execute(ctx, draft)
	return draft, err
}

func assertFailed(t *testing.T, got *lot.Lot, err, want error) {
	t.Helper()
	if !errors.Is(err, want) {
		t.Errorf("got error %v, want %v", err, want)
	}
	if !got.IsDefault() {
		t.Error("expected the default lot on error")
	}
}

func assertInternal(t *testing.T, got *lot.Lot, err error) {
	t.Helper()
	e, ok := exception.From(err)
	if !ok || e.Kind() != exception.KindInternal {
		t.Errorf("expected an internal error, got %v", err)
	}
	if !errors.Is(err, errInfra) {
		t.Error("expected the original cause to be kept")
	}
	if !got.IsDefault() {
		t.Error("expected the default lot on error")
	}
}

func TestCreateLotSuccess(t *testing.T) {
	s := newScenario(t)

	got, err := execute(s.useCase(), s.ctx(), s.draft())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Run("returns the created lot", func(t *testing.T) {
		if got.IsDefault() {
			t.Fatal("expected a real lot")
		}
		if got.FarmID() != s.farm.ID() || got.Code() != "LT-001" || got.Process() != lot.ProcessWashed {
			t.Error("unexpected lot data")
		}
		if !got.IsActive() || got.CurrentStage() != lot.StageNone || got.Version() != 1 {
			t.Error("expected an active lot without events and version 1")
		}
	})

	t.Run("the custodian is the producer of the farm", func(t *testing.T) {
		if got.CustodianID() != s.farm.ProducerID() {
			t.Errorf("got custodian %s, want %s", got.CustodianID(), s.farm.ProducerID())
		}
	})

	t.Run("checks the trimmed code", func(t *testing.T) {
		if s.lots.checkedCode != "LT-001" {
			t.Errorf("got %q, want %q", s.lots.checkedCode, "LT-001")
		}
	})

	t.Run("saves the lot", func(t *testing.T) {
		if len(s.lots.saved) != 1 || s.lots.saved[0].ID() != got.ID() {
			t.Error("expected the created lot to be saved once")
		}
	})

	t.Run("asks ABAC with the farm as resource", func(t *testing.T) {
		a := s.authorizer
		if a.action != ports.ActionCreateLot {
			t.Errorf("action: got %v, want %v", a.action, ports.ActionCreateLot)
		}
		if a.subject.ActorID != s.producer.ID() || a.subject.Role != actor.TypeProducer {
			t.Error("unexpected subject")
		}
		if a.resource.Type != ports.ResourceFarm || a.resource.ID != s.farm.ID() || a.resource.OwnerID != s.farm.ProducerID() {
			t.Error("unexpected resource")
		}
	})

	t.Run("publishes LotCreated", func(t *testing.T) {
		if len(s.publisher.events) != 1 {
			t.Fatalf("got %d events, want 1", len(s.publisher.events))
		}
		e := s.publisher.events[0]
		if e.Type != ports.EventLotCreated || e.AggregateType != ports.AggregateLot {
			t.Errorf("unexpected event type %v / %v", e.Type, e.AggregateType)
		}
		if e.AggregateID != got.ID() || e.Version != got.Version() {
			t.Error("unexpected aggregate ID or version")
		}
		if e.ActorID != s.producer.ID() || e.ActorRole != actor.TypeProducer.String() {
			t.Error("unexpected actor in event")
		}
		if e.CorrelationID != "req-123" {
			t.Errorf("got correlation %q, want %q", e.CorrelationID, "req-123")
		}
		if e.ID == uuid.Nil || e.OccurredAt.IsZero() {
			t.Error("expected event ID and date")
		}
		payload, ok := e.Payload.(createlot.LotCreatedPayload)
		if !ok {
			t.Fatalf("unexpected payload type %T", e.Payload)
		}
		if payload.LotID != got.ID() || payload.Code != "LT-001" || payload.Status != lot.StatusActive.String() {
			t.Error("unexpected payload data")
		}
	})
}

func TestCreateLotActorErrors(t *testing.T) {
	t.Run("no actor in the context", func(t *testing.T) {
		s := newScenario(t)
		got, err := execute(s.useCase(), context.Background(), s.draft())
		assertFailed(t, got, err, exception.New(exception.CodeActorNotFound))
	})

	t.Run("inactive actor", func(t *testing.T) {
		s := newScenario(t)
		inactive := actor.Restore(s.producer.ID(), "user_01", actor.TypeProducer, "Finca La Esperanza SAS",
			actor.StatusInactive, time.Now())
		s.actors.byID[s.producer.ID()] = inactive

		got, err := execute(s.useCase(), s.ctx(), s.draft())
		assertFailed(t, got, err, exception.New(exception.CodeActorNotFound))
	})

	t.Run("actor repository fails", func(t *testing.T) {
		s := newScenario(t)
		s.actors.err = errInfra
		got, err := execute(s.useCase(), s.ctx(), s.draft())
		assertInternal(t, got, err)
	})
}

func TestCreateLotFarmErrors(t *testing.T) {
	t.Run("farm not found", func(t *testing.T) {
		s := newScenario(t)
		draft := lot.Draft(uuid.New(), "LT-001", "Castillo", lot.ProcessWashed, time.Now().AddDate(0, -1, 0))
		got, err := execute(s.useCase(), s.ctx(), draft)
		assertFailed(t, got, err, exception.New(exception.CodeFarmNotFound))
	})

	t.Run("farm repository fails", func(t *testing.T) {
		s := newScenario(t)
		s.farms.err = errInfra
		got, err := execute(s.useCase(), s.ctx(), s.draft())
		assertInternal(t, got, err)
	})
}

func TestCreateLotAuthorization(t *testing.T) {
	t.Run("ABAC denies: forbidden, AccessDenied published, nothing saved", func(t *testing.T) {
		s := newScenario(t)
		s.authorizer.allowed = false

		got, err := execute(s.useCase(), s.ctx(), s.draft())
		assertFailed(t, got, err, exception.New(exception.CodeForbidden))

		e, _ := exception.From(err)
		if e.Kind() != exception.KindForbidden {
			t.Errorf("Kind: got %v, want %v", e.Kind(), exception.KindForbidden)
		}
		if len(s.publisher.events) != 1 || s.publisher.events[0].Type != ports.EventAccessDenied {
			t.Error("expected exactly one AccessDenied event")
		}
		if len(s.lots.saved) != 0 {
			t.Error("nothing must be saved when access is denied")
		}
	})

	t.Run("ABAC fails", func(t *testing.T) {
		s := newScenario(t)
		s.authorizer.err = errInfra
		got, err := execute(s.useCase(), s.ctx(), s.draft())
		assertInternal(t, got, err)
	})

	t.Run("publishing AccessDenied fails", func(t *testing.T) {
		s := newScenario(t)
		s.authorizer.allowed = false
		s.publisher.err = errInfra
		got, err := execute(s.useCase(), s.ctx(), s.draft())
		assertInternal(t, got, err)
	})
}

func TestCreateLotCodeErrors(t *testing.T) {
	t.Run("code already exists in the farm", func(t *testing.T) {
		s := newScenario(t)
		s.lots.existingCode = "LT-001"

		got, err := execute(s.useCase(), s.ctx(), s.draft())
		assertFailed(t, got, err, exception.New(exception.CodeLotCodeAlreadyExists))
		if len(s.lots.saved) != 0 {
			t.Error("nothing must be saved when the code exists")
		}
	})

	t.Run("checking the code fails", func(t *testing.T) {
		s := newScenario(t)
		s.lots.existsErr = errInfra
		got, err := execute(s.useCase(), s.ctx(), s.draft())
		assertInternal(t, got, err)
	})
}

func TestCreateLotDomainErrors(t *testing.T) {
	s := newScenario(t)
	harvest := time.Now().AddDate(0, -1, 0)

	tests := []struct {
		name  string
		draft *lot.Lot
		want  error
	}{
		{"code too short", lot.Draft(s.farm.ID(), "LT", "Castillo", lot.ProcessWashed, harvest), exception.New(exception.CodeLotInvalidCode)},
		{"empty variety", lot.Draft(s.farm.ID(), "LT-001", "  ", lot.ProcessWashed, harvest), exception.New(exception.CodeLotInvalidVariety)},
		{"unknown process", lot.Draft(s.farm.ID(), "LT-001", "Castillo", lot.ProcessUnknown, harvest), exception.New(exception.CodeLotInvalidProcess)},
		{"future harvest date", lot.Draft(s.farm.ID(), "LT-001", "Castillo", lot.ProcessWashed, time.Now().AddDate(0, 0, 1)), exception.New(exception.CodeLotInvalidHarvestDate)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lots := &fakeLots{}
			uc := createlot.New(s.actors, s.farms, lots, &fakeAuthorizer{allowed: true}, &fakePublisher{})

			got, err := execute(uc, s.ctx(), tt.draft)
			assertFailed(t, got, err, tt.want)
			if len(lots.saved) != 0 {
				t.Error("nothing must be saved when the domain rejects the lot")
			}
		})
	}
}

func TestCreateLotPersistenceErrors(t *testing.T) {
	t.Run("saving fails: nothing published", func(t *testing.T) {
		s := newScenario(t)
		s.lots.saveErr = errInfra

		got, err := execute(s.useCase(), s.ctx(), s.draft())
		assertInternal(t, got, err)
		if len(s.publisher.events) != 0 {
			t.Error("no event must be published when saving fails")
		}
	})

	t.Run("publishing LotCreated fails", func(t *testing.T) {
		s := newScenario(t)
		s.publisher.err = errInfra
		got, err := execute(s.useCase(), s.ctx(), s.draft())
		assertInternal(t, got, err)
	})
}

func TestCreateLotNilDraft(t *testing.T) {
	s := newScenario(t)

	got, err := execute(s.useCase(), s.ctx(), nil)
	assertFailed(t, got, err, exception.New(exception.CodeFarmNotFound))
}
