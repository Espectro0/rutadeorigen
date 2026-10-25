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

type txKey struct{}

func inTx(ctx context.Context) bool {
	v, _ := ctx.Value(txKey{}).(bool)
	return v
}

type fakeTxManager struct {
	err   error
	calls int
}

func (m *fakeTxManager) WithinTx(ctx context.Context, fn func(context.Context) error) error {
	m.calls++
	if m.err != nil {
		return m.err
	}
	return fn(context.WithValue(ctx, txKey{}, true))
}

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

func (r fakeFarms) Save(_ context.Context, _ *farm.Farm) error {
	return nil
}

func (r fakeFarms) ListByProducer(_ context.Context, _ uuid.UUID) ([]*farm.Farm, error) {
	return []*farm.Farm{}, nil
}

type fakeLots struct {
	existingCode string
	existsErr    error
	saveErr      error
	checkedCode  string
	saved        []*lot.Lot
	savedInTx    []bool
}

func (r *fakeLots) ExistsByCode(_ context.Context, _ uuid.UUID, code string) (bool, error) {
	r.checkedCode = code
	if r.existsErr != nil {
		return false, r.existsErr
	}
	return code == r.existingCode, nil
}

func (r *fakeLots) FindByID(_ context.Context, _ uuid.UUID) (*lot.Lot, error) {
	return lot.Default(), nil
}

func (r *fakeLots) List(_ context.Context, _ lot.Filter, _ lot.Cursor, _ int) ([]*lot.Lot, error) {
	return nil, nil
}

func (r *fakeLots) UpdateStage(_ context.Context, _ *lot.Lot, _ int) error {
	return nil
}

func (r *fakeLots) Save(ctx context.Context, l *lot.Lot) error {
	if r.saveErr != nil {
		return r.saveErr
	}
	r.saved = append(r.saved, l)
	r.savedInTx = append(r.savedInTx, inTx(ctx))
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
	inTx   []bool
}

func (p *fakePublisher) Publish(ctx context.Context, e ports.Event) error {
	if p.err != nil {
		return p.err
	}
	p.events = append(p.events, e)
	p.inTx = append(p.inTx, inTx(ctx))
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
	tx         *fakeTxManager
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
		tx:         &fakeTxManager{},
	}
}

func (s *scenario) useCase() *createlot.CreateLot {
	return createlot.New(s.actors, s.farms, s.lots, s.authorizer, s.publisher, s.tx)
}

func (s *scenario) ctx() context.Context {
	ctx := requestcontext.WithActorID(context.Background(), s.producer.ID())
	return requestcontext.WithCorrelationID(ctx, "req-123")
}

func (s *scenario) draft() *lot.Lot {
	return lot.Draft(s.farm.ID(), "  LT-001  ", "Castillo", lot.ProcessWashed, time.Now().AddDate(0, -1, 0))
}

func execute(ctx context.Context, uc createlot.UseCase, draft *lot.Lot) (*lot.Lot, error) {
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

	got, err := execute(s.ctx(), s.useCase(), s.draft())
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
		if payload.CustodianID != got.CustodianID() {
			t.Errorf("custodian: got %s, want %s", payload.CustodianID, got.CustodianID())
		}
	})

	t.Run("LotCreated carries a snapshot of the farm", func(t *testing.T) {
		payload := s.publisher.events[0].Payload.(createlot.LotCreatedPayload)
		want := createlot.FarmSnapshot{ID: s.farm.ID(), Name: "La Esperanza", Altitude: 1650}
		if payload.Farm != want {
			t.Errorf("farm: got %+v, want %+v", payload.Farm, want)
		}
	})

	t.Run("LotCreated carries a snapshot of the producer", func(t *testing.T) {
		payload := s.publisher.events[0].Payload.(createlot.LotCreatedPayload)
		want := createlot.ProducerSnapshot{ID: s.producer.ID(), Name: "Finca La Esperanza SAS"}
		if payload.Producer != want {
			t.Errorf("producer: got %+v, want %+v", payload.Producer, want)
		}
	})
}

func TestCreateLotProducerSnapshot(t *testing.T) {
	t.Run("the producer comes from the farm, not from the requester", func(t *testing.T) {
		s := newScenario(t)
		org, err := actor.New("user_02", actor.TypeOrganization, "Cooperativa de Caficultores")
		if err != nil {
			t.Fatal(err)
		}
		s.actors.byID[org.ID()] = org
		ctx := requestcontext.WithActorID(context.Background(), org.ID())

		if _, err := execute(ctx, s.useCase(), s.draft()); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		e := s.publisher.events[0]
		payload := e.Payload.(createlot.LotCreatedPayload)
		if payload.Producer.ID != s.producer.ID() || payload.Producer.Name != "Finca La Esperanza SAS" {
			t.Errorf("producer: got %+v, want the farm producer", payload.Producer)
		}
		if e.ActorID != org.ID() || e.ActorRole != actor.TypeOrganization.String() {
			t.Error("expected the requester as the event actor")
		}
	})

	t.Run("the producer of the farm is inactive: nothing published", func(t *testing.T) {
		s := newScenario(t)
		org, err := actor.New("user_02", actor.TypeOrganization, "Cooperativa de Caficultores")
		if err != nil {
			t.Fatal(err)
		}
		s.actors.byID[org.ID()] = org
		s.actors.byID[s.producer.ID()] = actor.Restore(s.producer.ID(), "user_01", actor.TypeProducer,
			"Finca La Esperanza SAS", actor.StatusInactive, time.Now())
		ctx := requestcontext.WithActorID(context.Background(), org.ID())

		got, err := execute(ctx, s.useCase(), s.draft())
		assertFailed(t, got, err, exception.New(exception.CodeActorNotFound))
		if len(s.publisher.events) != 0 {
			t.Error("no event must be published when the producer is inactive")
		}
	})

	t.Run("the producer of the farm does not exist: nothing published", func(t *testing.T) {
		s := newScenario(t)
		org, err := actor.New("user_02", actor.TypeOrganization, "Cooperativa de Caficultores")
		if err != nil {
			t.Fatal(err)
		}
		s.actors.byID = map[uuid.UUID]*actor.Actor{org.ID(): org}
		ctx := requestcontext.WithActorID(context.Background(), org.ID())

		got, err := execute(ctx, s.useCase(), s.draft())
		assertFailed(t, got, err, exception.New(exception.CodeActorNotFound))
		if len(s.publisher.events) != 0 {
			t.Error("no event must be published when the producer does not exist")
		}
	})
}

func TestCreateLotActorErrors(t *testing.T) {
	t.Run("no actor in the context", func(t *testing.T) {
		s := newScenario(t)
		got, err := execute(context.Background(), s.useCase(), s.draft())
		assertFailed(t, got, err, exception.New(exception.CodeActorNotFound))
	})

	t.Run("inactive actor", func(t *testing.T) {
		s := newScenario(t)
		inactive := actor.Restore(s.producer.ID(), "user_01", actor.TypeProducer, "Finca La Esperanza SAS",
			actor.StatusInactive, time.Now())
		s.actors.byID[s.producer.ID()] = inactive

		got, err := execute(s.ctx(), s.useCase(), s.draft())
		assertFailed(t, got, err, exception.New(exception.CodeActorNotFound))
	})

	t.Run("actor repository fails", func(t *testing.T) {
		s := newScenario(t)
		s.actors.err = errInfra
		got, err := execute(s.ctx(), s.useCase(), s.draft())
		assertInternal(t, got, err)
	})
}

func TestCreateLotFarmErrors(t *testing.T) {
	t.Run("farm not found", func(t *testing.T) {
		s := newScenario(t)
		draft := lot.Draft(uuid.New(), "LT-001", "Castillo", lot.ProcessWashed, time.Now().AddDate(0, -1, 0))
		got, err := execute(s.ctx(), s.useCase(), draft)
		assertFailed(t, got, err, exception.New(exception.CodeFarmNotFound))
	})

	t.Run("farm repository fails", func(t *testing.T) {
		s := newScenario(t)
		s.farms.err = errInfra
		got, err := execute(s.ctx(), s.useCase(), s.draft())
		assertInternal(t, got, err)
	})
}

func TestCreateLotAuthorization(t *testing.T) {
	t.Run("ABAC denies: forbidden, AccessDenied published, nothing saved", func(t *testing.T) {
		s := newScenario(t)
		s.authorizer.allowed = false

		got, err := execute(s.ctx(), s.useCase(), s.draft())
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
		got, err := execute(s.ctx(), s.useCase(), s.draft())
		assertInternal(t, got, err)
	})

	t.Run("publishing AccessDenied fails", func(t *testing.T) {
		s := newScenario(t)
		s.authorizer.allowed = false
		s.publisher.err = errInfra
		got, err := execute(s.ctx(), s.useCase(), s.draft())
		assertInternal(t, got, err)
	})
}

func TestCreateLotCodeErrors(t *testing.T) {
	t.Run("code already exists in the farm", func(t *testing.T) {
		s := newScenario(t)
		s.lots.existingCode = "LT-001"

		got, err := execute(s.ctx(), s.useCase(), s.draft())
		assertFailed(t, got, err, exception.New(exception.CodeLotCodeAlreadyExists))
		if len(s.lots.saved) != 0 {
			t.Error("nothing must be saved when the code exists")
		}
	})

	t.Run("checking the code fails", func(t *testing.T) {
		s := newScenario(t)
		s.lots.existsErr = errInfra
		got, err := execute(s.ctx(), s.useCase(), s.draft())
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
			uc := createlot.New(s.actors, s.farms, lots, &fakeAuthorizer{allowed: true}, &fakePublisher{}, &fakeTxManager{})

			got, err := execute(s.ctx(), uc, tt.draft)
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

		got, err := execute(s.ctx(), s.useCase(), s.draft())
		assertInternal(t, got, err)
		if len(s.publisher.events) != 0 {
			t.Error("no event must be published when saving fails")
		}
	})

	t.Run("publishing LotCreated fails", func(t *testing.T) {
		s := newScenario(t)
		s.publisher.err = errInfra
		got, err := execute(s.ctx(), s.useCase(), s.draft())
		assertInternal(t, got, err)
	})
}

func TestCreateLotNilDraft(t *testing.T) {
	s := newScenario(t)

	got, err := execute(s.ctx(), s.useCase(), nil)
	assertFailed(t, got, err, exception.New(exception.CodeFarmNotFound))
}

func TestCreateLotTransaction(t *testing.T) {
	t.Run("saves the lot and publishes LotCreated in one transaction", func(t *testing.T) {
		s := newScenario(t)

		if _, err := execute(s.ctx(), s.useCase(), s.draft()); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if s.tx.calls != 1 {
			t.Errorf("got %d transactions, want 1", s.tx.calls)
		}
		if len(s.lots.savedInTx) != 1 || !s.lots.savedInTx[0] {
			t.Error("expected the lot to be saved inside the transaction")
		}
		if len(s.publisher.inTx) != 1 || !s.publisher.inTx[0] {
			t.Error("expected LotCreated to be published inside the transaction")
		}
	})

	t.Run("the transaction fails: internal error, nothing saved or published", func(t *testing.T) {
		s := newScenario(t)
		s.tx.err = errInfra

		got, err := execute(s.ctx(), s.useCase(), s.draft())
		assertInternal(t, got, err)
		if len(s.lots.saved) != 0 || len(s.publisher.events) != 0 {
			t.Error("nothing must be saved or published when the transaction fails")
		}
	})

	t.Run("a domain error inside the transaction keeps its code", func(t *testing.T) {
		s := newScenario(t)
		s.lots.saveErr = exception.New(exception.CodeLotCodeAlreadyExists)

		got, err := execute(s.ctx(), s.useCase(), s.draft())
		assertFailed(t, got, err, exception.New(exception.CodeLotCodeAlreadyExists))
	})

	t.Run("AccessDenied is published outside a transaction", func(t *testing.T) {
		s := newScenario(t)
		s.authorizer.allowed = false

		_, _ = execute(s.ctx(), s.useCase(), s.draft())
		if s.tx.calls != 0 {
			t.Errorf("got %d transactions, want 0", s.tx.calls)
		}
		if len(s.publisher.inTx) != 1 || s.publisher.inTx[0] {
			t.Error("expected AccessDenied to be published outside a transaction")
		}
	})

	t.Run("a duplicated code does not open a transaction", func(t *testing.T) {
		s := newScenario(t)
		s.lots.existingCode = "LT-001"

		_, _ = execute(s.ctx(), s.useCase(), s.draft())
		if s.tx.calls != 0 {
			t.Errorf("got %d transactions, want 0", s.tx.calls)
		}
	})
}
