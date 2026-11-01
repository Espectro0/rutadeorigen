package transferlot_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"rutadeorigen/lib/requestcontext"
	"rutadeorigen/traceability/internal/application/output/ports"
	"rutadeorigen/traceability/internal/application/usecases/access"
	"rutadeorigen/traceability/internal/application/usecases/transferlot"
	"rutadeorigen/traceability/internal/domain/actor"
	"rutadeorigen/traceability/internal/domain/lot"
	"rutadeorigen/traceability/internal/platform/exception"
	"rutadeorigen/traceability/internal/platform/helper"
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

func (r fakeActors) FindByUserID(_ context.Context, _ string) (*actor.Actor, error) {
	return actor.Default(), nil
}

func (r fakeActors) ListByType(_ context.Context, _ actor.Type) ([]*actor.Actor, error) {
	return nil, nil
}

func (r fakeActors) Save(_ context.Context, _ *actor.Actor) error {
	return nil
}

type custodyCall struct {
	custodianID     uuid.UUID
	version         int
	expectedVersion int
	inTx            bool
}

type fakeLots struct {
	byID      map[uuid.UUID]*lot.Lot
	findErr   error
	updateErr error
	updates   []custodyCall
}

func (r *fakeLots) ExistsByCode(_ context.Context, _ uuid.UUID, _ string) (bool, error) {
	return false, nil
}

func (r *fakeLots) Save(_ context.Context, _ *lot.Lot) error {
	return nil
}

func (r *fakeLots) List(_ context.Context, _ lot.Filter, _ lot.Cursor, _ int) ([]*lot.Lot, error) {
	return nil, nil
}

func (r *fakeLots) UpdateStage(_ context.Context, _ *lot.Lot, _ int) error {
	return nil
}

func (r *fakeLots) FindByID(_ context.Context, id uuid.UUID) (*lot.Lot, error) {
	if r.findErr != nil {
		return lot.Default(), r.findErr
	}
	if l, ok := r.byID[id]; ok {
		return l, nil
	}
	return lot.Default(), nil
}

func (r *fakeLots) UpdateStatus(_ context.Context, _ *lot.Lot, _ int) error { return nil }

func (r *fakeLots) UpdateCustodian(ctx context.Context, l *lot.Lot, expectedVersion int) error {
	if r.updateErr != nil {
		return r.updateErr
	}
	r.updates = append(r.updates, custodyCall{
		custodianID:     l.CustodianID(),
		version:         l.Version(),
		expectedVersion: expectedVersion,
		inTx:            inTx(ctx),
	})
	return nil
}

type fakeEvents struct {
	findErr error
	saveErr error
	saved   []*lot.Event
	savedTx []bool
}

func (r *fakeEvents) FindLast(_ context.Context, lotID uuid.UUID) (*lot.Event, error) {
	if r.findErr != nil {
		return lot.DefaultEvent(), r.findErr
	}
	for i := len(r.saved) - 1; i >= 0; i-- {
		if r.saved[i].LotID() == lotID {
			return r.saved[i], nil
		}
	}
	return lot.DefaultEvent(), nil
}

func (r *fakeEvents) ListByLotID(_ context.Context, _ uuid.UUID) ([]*lot.Event, error) {
	return r.saved, nil
}

func (r *fakeEvents) Save(ctx context.Context, e *lot.Event) error {
	if r.saveErr != nil {
		return r.saveErr
	}
	r.saved = append(r.saved, e)
	r.savedTx = append(r.savedTx, inTx(ctx))
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
	errFor ports.EventType
	events []ports.Event
	inTx   []bool
}

func (p *fakePublisher) Publish(ctx context.Context, e ports.Event) error {
	if p.err != nil && (p.errFor == "" || p.errFor == e.Type) {
		return p.err
	}
	p.events = append(p.events, e)
	p.inTx = append(p.inTx, inTx(ctx))
	return nil
}

type scenario struct {
	producer   *actor.Actor
	roaster    *actor.Actor
	brand      *actor.Actor
	lot        *lot.Lot
	actors     fakeActors
	lots       *fakeLots
	events     *fakeEvents
	authorizer *fakeAuthorizer
	publisher  *fakePublisher
	tx         *fakeTxManager
}

func newActor(t *testing.T, userID string, actorType actor.Type) *actor.Actor {
	t.Helper()

	a, err := actor.New(userID, actorType, "Actor "+userID, userID+"@example.com")
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func newScenario(t *testing.T) *scenario {
	t.Helper()

	producer := newActor(t, "user_01", actor.TypeProducer)
	roaster := newActor(t, "user_02", actor.TypeRoaster)
	brand := newActor(t, "user_03", actor.TypeBrand)

	harvest := helper.DateUTC(time.Now().AddDate(0, -1, 0))
	l, err := lot.New(uuid.New(), producer.ID(), "LT-001", "Castillo", lot.ProcessWashed, harvest)
	if err != nil {
		t.Fatal(err)
	}
	if err := l.AdvanceTo(lot.StageFarm); err != nil {
		t.Fatal(err)
	}

	return &scenario{
		producer: producer,
		roaster:  roaster,
		brand:    brand,
		lot:      l,
		actors: fakeActors{byID: map[uuid.UUID]*actor.Actor{
			producer.ID(): producer,
			roaster.ID():  roaster,
			brand.ID():    brand,
		}},
		lots:       &fakeLots{byID: map[uuid.UUID]*lot.Lot{l.ID(): l}},
		events:     &fakeEvents{},
		authorizer: &fakeAuthorizer{allowed: true},
		publisher:  &fakePublisher{},
		tx:         &fakeTxManager{},
	}
}

func (s *scenario) ctx() context.Context {
	ctx := requestcontext.WithActorID(context.Background(), s.producer.ID())
	return requestcontext.WithCorrelationID(ctx, "req-123")
}

func (s *scenario) command() transferlot.Command {
	return transferlot.Command{
		LotID:           s.lot.ID(),
		ExpectedVersion: s.lot.Version(),
		TargetActorID:   s.roaster.ID(),
		Note:            "  Handed over  ",
	}
}

func (s *scenario) execute(cmd transferlot.Command) (transferlot.Result, error) {
	uc := transferlot.New(s.actors, s.lots, s.events, s.authorizer, s.publisher, s.tx)
	return uc.Execute(s.ctx(), cmd)
}

func (s *scenario) assertNothingPersisted(t *testing.T) {
	t.Helper()
	if len(s.lots.updates) != 0 {
		t.Error("the lot must not be updated")
	}
	if len(s.events.saved) != 0 {
		t.Error("no lot event must be saved")
	}
	for _, e := range s.publisher.events {
		if e.Type == ports.EventLotTransferred {
			t.Error("no transfer event must be published")
		}
	}
}

func assertFailed(t *testing.T, got transferlot.Result, err error, want exception.Code) {
	t.Helper()
	if !exception.HasCode(err, want) {
		t.Errorf("got error %v, want code %q", err, want)
	}
	assertDefaultResult(t, got)
}

func assertInternal(t *testing.T, got transferlot.Result, err error) {
	t.Helper()
	e, ok := exception.From(err)
	if !ok || e.Kind() != exception.KindInternal {
		t.Errorf("expected an internal error, got %v", err)
	}
	if !errors.Is(err, errInfra) {
		t.Error("expected the original cause to be kept")
	}
	assertDefaultResult(t, got)
}

func assertDefaultResult(t *testing.T, got transferlot.Result) {
	t.Helper()
	if !got.Event.IsDefault() {
		t.Error("expected the default event on error")
	}
	if got.LotVersion != 0 {
		t.Errorf("got version %d, want 0 on error", got.LotVersion)
	}
}

func TestTransferLotSuccess(t *testing.T) {
	s := newScenario(t)
	cmd := s.command()

	got, err := s.execute(cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Run("returns the transfer event chained from the genesis hash", func(t *testing.T) {
		e := got.Event
		if e.IsDefault() || e.Type() != lot.EventTypeTransfer || e.Stage() != lot.StageFarm {
			t.Fatalf("unexpected event: %+v", e)
		}
		if e.LotID() != s.lot.ID() || e.ActorID() != s.producer.ID() || e.ActorRole() != actor.TypeProducer.String() {
			t.Error("unexpected lot or actor")
		}
		if e.TargetActorID() != s.roaster.ID() {
			t.Errorf("target: got %v, want %v", e.TargetActorID(), s.roaster.ID())
		}
		if e.Sequence() != 1 || e.PreviousHash() != lot.GenesisHash || !e.VerifyHash() {
			t.Error("expected the first event of a valid chain")
		}
		if e.Note() != "Handed over" || e.Details() != nil {
			t.Errorf("unexpected note %q or details %v", e.Note(), e.Details())
		}
	})

	t.Run("moves the custody and bumps the version", func(t *testing.T) {
		if s.lot.CustodianID() != s.roaster.ID() || got.LotVersion != cmd.ExpectedVersion+1 {
			t.Errorf("got custodian %v and version %d", s.lot.CustodianID(), got.LotVersion)
		}
		if len(s.lots.updates) != 1 {
			t.Fatalf("got %d updates, want 1", len(s.lots.updates))
		}
		u := s.lots.updates[0]
		if u.custodianID != s.roaster.ID() || u.version != got.LotVersion || u.expectedVersion != cmd.ExpectedVersion || !u.inTx {
			t.Errorf("unexpected update %+v", u)
		}
	})

	t.Run("saves the event inside the transaction", func(t *testing.T) {
		if len(s.events.saved) != 1 || s.events.saved[0] != got.Event || !s.events.savedTx[0] {
			t.Error("expected the event saved once inside the transaction")
		}
		if s.tx.calls != 1 {
			t.Errorf("got %d transactions, want 1", s.tx.calls)
		}
	})

	t.Run("asks ABAC with the lot as resource", func(t *testing.T) {
		a := s.authorizer
		if a.action != ports.ActionTransferLot {
			t.Errorf("action: got %v, want %v", a.action, ports.ActionTransferLot)
		}
		if a.subject.ActorID != s.producer.ID() || a.subject.Role != actor.TypeProducer {
			t.Error("unexpected subject")
		}
		if a.resource.Type != ports.ResourceLot || a.resource.ID != s.lot.ID() || a.resource.CustodianID != s.producer.ID() {
			t.Errorf("unexpected resource %+v", a.resource)
		}
	})

	t.Run("publishes LotTransferred inside the transaction", func(t *testing.T) {
		if len(s.publisher.events) != 1 || !s.publisher.inTx[0] {
			t.Fatal("expected one event published inside the transaction")
		}
		event := s.publisher.events[0]
		if event.Type != ports.EventLotTransferred || event.AggregateType != ports.AggregateLot ||
			event.AggregateID != s.lot.ID() || event.Version != got.LotVersion {
			t.Errorf("unexpected envelope %+v", event)
		}
		if event.ActorID != s.producer.ID() || event.CorrelationID != "req-123" {
			t.Error("unexpected actor or correlation id")
		}
		payload, ok := event.Payload.(transferlot.LotTransferredPayload)
		if !ok {
			t.Fatalf("unexpected payload type %T", event.Payload)
		}
		if payload.LotID != s.lot.ID() || payload.FromActorID != s.producer.ID() || payload.CustodianID != s.roaster.ID() ||
			payload.Stage != "farm" || payload.Status != "active" || payload.EventID != got.Event.ID() || payload.Sequence != 1 {
			t.Errorf("unexpected payload %+v", payload)
		}
	})
}

func TestTransferLotChainsAfterThePreviousEvent(t *testing.T) {
	s := newScenario(t)
	previous := &fakeEvents{}
	first, err := lot.NewTransferEvent(lot.TransferEventParams{
		LotID: s.lot.ID(), Stage: lot.StageFarm, EventDate: time.Now(), ActorID: s.producer.ID(),
		ActorRole: "producer", TargetActorID: s.roaster.ID(),
	})
	if err != nil {
		t.Fatal(err)
	}
	previous.saved = append(previous.saved, first)
	s.events = previous

	got, err := s.execute(s.command())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.Event.Sequence() != 2 || got.Event.PreviousHash() != first.Hash() {
		t.Errorf("got sequence %d and previous hash %q", got.Event.Sequence(), got.Event.PreviousHash())
	}
}

func TestTransferLotToABrandAfterRoasting(t *testing.T) {
	s := newScenario(t)
	if err := s.lot.TransferTo(s.roaster.ID(), actor.TypeRoaster); err != nil {
		t.Fatal(err)
	}
	if err := s.lot.AdvanceTo(lot.StageRoasting); err != nil {
		t.Fatal(err)
	}
	s.producer = s.roaster
	cmd := s.command()
	cmd.TargetActorID = s.brand.ID()

	got, err := s.execute(cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if s.lot.CustodianID() != s.brand.ID() || got.Event.Stage() != lot.StageRoasting {
		t.Error("expected the lot handed to the brand at the roasting stage")
	}
}

func TestTransferLotActorErrors(t *testing.T) {
	t.Run("the requester does not exist", func(t *testing.T) {
		s := newScenario(t)
		s.actors = fakeActors{byID: map[uuid.UUID]*actor.Actor{}}

		got, err := s.execute(s.command())
		assertFailed(t, got, err, exception.CodeActorNotFound)
		s.assertNothingPersisted(t)
	})

	t.Run("the requester lookup fails", func(t *testing.T) {
		s := newScenario(t)
		s.actors.err = errInfra

		got, err := s.execute(s.command())
		assertInternal(t, got, err)
	})

	t.Run("the target does not exist", func(t *testing.T) {
		s := newScenario(t)
		cmd := s.command()
		cmd.TargetActorID = uuid.New()

		got, err := s.execute(cmd)
		assertFailed(t, got, err, exception.CodeActorNotFound)
		s.assertNothingPersisted(t)
	})

	t.Run("the target is inactive", func(t *testing.T) {
		s := newScenario(t)
		inactive := actor.Restore(s.roaster.ID(), s.roaster.UserID(), actor.TypeRoaster, s.roaster.Name(),
			s.roaster.Email(), actor.StatusInactive, s.roaster.CreatedAt())
		s.actors.byID[inactive.ID()] = inactive

		got, err := s.execute(s.command())
		assertFailed(t, got, err, exception.CodeActorNotFound)
		s.assertNothingPersisted(t)
	})
}

func TestTransferLotLotErrors(t *testing.T) {
	t.Run("the lot does not exist", func(t *testing.T) {
		s := newScenario(t)
		cmd := s.command()
		cmd.LotID = uuid.New()

		got, err := s.execute(cmd)
		assertFailed(t, got, err, exception.CodeLotNotFound)
	})

	t.Run("the lot lookup fails", func(t *testing.T) {
		s := newScenario(t)
		s.lots.findErr = errInfra

		got, err := s.execute(s.command())
		assertInternal(t, got, err)
	})

	t.Run("the version is stale", func(t *testing.T) {
		s := newScenario(t)
		cmd := s.command()
		cmd.ExpectedVersion = 99

		got, err := s.execute(cmd)
		assertFailed(t, got, err, exception.CodeLotVersionConflict)
		s.assertNothingPersisted(t)
	})
}

func TestTransferLotDomainErrors(t *testing.T) {
	t.Run("the lot has no stage yet", func(t *testing.T) {
		s := newScenario(t)
		fresh, err := lot.New(uuid.New(), s.producer.ID(), "LT-002", "Castillo", lot.ProcessWashed, helper.DateUTC(time.Now().AddDate(0, -1, 0)))
		if err != nil {
			t.Fatal(err)
		}
		s.lots.byID[fresh.ID()] = fresh
		cmd := s.command()
		cmd.LotID, cmd.ExpectedVersion = fresh.ID(), fresh.Version()

		got, err := s.execute(cmd)
		assertFailed(t, got, err, exception.CodeLotNotTransferable)
		s.assertNothingPersisted(t)
	})

	t.Run("the lot is voided", func(t *testing.T) {
		s := newScenario(t)
		if err := s.lot.Void("Spoiled"); err != nil {
			t.Fatal(err)
		}
		cmd := s.command()

		got, err := s.execute(cmd)
		assertFailed(t, got, err, exception.CodeLotVoided)
		s.assertNothingPersisted(t)
	})

	t.Run("the target has the wrong role", func(t *testing.T) {
		s := newScenario(t)
		cmd := s.command()
		cmd.TargetActorID = s.brand.ID()

		got, err := s.execute(cmd)
		assertFailed(t, got, err, exception.CodeLotInvalidTransferTarget)
		s.assertNothingPersisted(t)
	})

	t.Run("the note is too long", func(t *testing.T) {
		s := newScenario(t)
		cmd := s.command()
		note := make([]rune, 501)
		for i := range note {
			note[i] = 'a'
		}
		cmd.Note = string(note)

		got, err := s.execute(cmd)
		assertFailed(t, got, err, exception.CodeLotInvalidNote)
		s.assertNothingPersisted(t)
	})
}

func TestTransferLotAuthorization(t *testing.T) {
	t.Run("ABAC denies: forbidden, AccessDenied published, nothing persisted", func(t *testing.T) {
		s := newScenario(t)
		s.authorizer.allowed = false

		got, err := s.execute(s.command())
		assertFailed(t, got, err, exception.CodeForbidden)

		if len(s.publisher.events) != 1 || s.publisher.events[0].Type != ports.EventAccessDenied {
			t.Fatal("expected exactly one AccessDenied event")
		}
		payload, ok := s.publisher.events[0].Payload.(access.DeniedPayload)
		if !ok || payload.Action != string(ports.ActionTransferLot) || payload.ResourceID != s.lot.ID().String() {
			t.Errorf("unexpected payload %+v", s.publisher.events[0].Payload)
		}
		s.assertNothingPersisted(t)
		if s.tx.calls != 0 {
			t.Error("expected no transaction")
		}
	})

	t.Run("denied before revealing a stale version", func(t *testing.T) {
		s := newScenario(t)
		s.authorizer.allowed = false
		cmd := s.command()
		cmd.ExpectedVersion = 99

		got, err := s.execute(cmd)
		assertFailed(t, got, err, exception.CodeForbidden)
	})

	t.Run("ABAC fails", func(t *testing.T) {
		s := newScenario(t)
		s.authorizer.err = errInfra

		got, err := s.execute(s.command())
		assertInternal(t, got, err)
	})
}

func TestTransferLotPersistenceErrors(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*scenario)
	}{
		{"the transaction fails", func(s *scenario) { s.tx.err = errInfra }},
		{"the last event lookup fails", func(s *scenario) { s.events.findErr = errInfra }},
		{"the custody update fails", func(s *scenario) { s.lots.updateErr = errInfra }},
		{"the event save fails", func(s *scenario) { s.events.saveErr = errInfra }},
		{"publishing fails", func(s *scenario) {
			s.publisher.err = errInfra
			s.publisher.errFor = ports.EventLotTransferred
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newScenario(t)
			tt.setup(s)

			got, err := s.execute(s.command())
			assertInternal(t, got, err)
		})
	}
}

func TestTransferLotDoesNotOpenTransactionOnEarlyFailures(t *testing.T) {
	s := newScenario(t)
	cmd := s.command()
	cmd.TargetActorID = uuid.New()

	_, _ = s.execute(cmd)

	if s.tx.calls != 0 {
		t.Errorf("got %d transactions, want 0", s.tx.calls)
	}
}
