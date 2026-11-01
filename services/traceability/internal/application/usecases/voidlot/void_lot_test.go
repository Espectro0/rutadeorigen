package voidlot_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"rutadeorigen/lib/requestcontext"
	"rutadeorigen/traceability/internal/application/output/ports"
	"rutadeorigen/traceability/internal/application/usecases/access"
	"rutadeorigen/traceability/internal/application/usecases/voidlot"
	"rutadeorigen/traceability/internal/domain/actor"
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
	actor *actor.Actor
	err   error
}

func (f fakeActors) FindByID(_ context.Context, _ uuid.UUID) (*actor.Actor, error) {
	if f.err != nil {
		return actor.Default(), f.err
	}
	return f.actor, nil
}

func (f fakeActors) FindByUserID(_ context.Context, _ string) (*actor.Actor, error) {
	return actor.Default(), nil
}

func (f fakeActors) ListByType(_ context.Context, _ actor.Type) ([]*actor.Actor, error) {
	return nil, nil
}

func (f fakeActors) Save(_ context.Context, _ *actor.Actor) error { return nil }

type statusCall struct {
	status          lot.Status
	reason          string
	version         int
	expectedVersion int
	inTx            bool
}

type fakeLots struct {
	lot       *lot.Lot
	findErr   error
	updateErr error
	updates   []statusCall
}

func (f *fakeLots) ExistsByCode(_ context.Context, _ uuid.UUID, _ string) (bool, error) {
	return false, nil
}

func (f *fakeLots) Save(_ context.Context, _ *lot.Lot) error { return nil }

func (f *fakeLots) List(_ context.Context, _ lot.Filter, _ lot.Cursor, _ int) ([]*lot.Lot, error) {
	return nil, nil
}

func (f *fakeLots) UpdateStage(_ context.Context, _ *lot.Lot, _ int) error { return nil }

func (f *fakeLots) UpdateCustodian(_ context.Context, _ *lot.Lot, _ int) error { return nil }

func (f *fakeLots) FindByID(_ context.Context, _ uuid.UUID) (*lot.Lot, error) {
	if f.findErr != nil {
		return lot.Default(), f.findErr
	}
	return f.lot, nil
}

func (f *fakeLots) UpdateStatus(ctx context.Context, l *lot.Lot, expectedVersion int) error {
	if f.updateErr != nil {
		return f.updateErr
	}
	f.updates = append(f.updates, statusCall{
		status:          l.Status(),
		reason:          l.VoidReason(),
		version:         l.Version(),
		expectedVersion: expectedVersion,
		inTx:            inTx(ctx),
	})
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
	organization *actor.Actor
	lot          *lot.Lot
	actors       fakeActors
	lots         *fakeLots
	authorizer   *fakeAuthorizer
	publisher    *fakePublisher
	tx           *fakeTxManager
}

func newScenario(t *testing.T) *scenario {
	t.Helper()

	organization, err := actor.New("user_01", actor.TypeOrganization, "Ruta de Origen", "user_01@example.com")
	if err != nil {
		t.Fatal(err)
	}
	l := lot.Restore(lot.RestoreParams{
		ID:           uuid.New(),
		FarmID:       uuid.New(),
		CustodianID:  uuid.New(),
		Code:         "LOT-001",
		Variety:      "Castillo",
		Process:      lot.ProcessWashed,
		HarvestDate:  time.Date(2026, time.September, 15, 0, 0, 0, 0, time.UTC),
		CurrentStage: lot.StageRoasting,
		Status:       lot.StatusActive,
		Version:      4,
		CreatedAt:    time.Date(2026, time.September, 20, 14, 30, 0, 0, time.UTC),
	})

	return &scenario{
		organization: organization,
		lot:          l,
		actors:       fakeActors{actor: organization},
		lots:         &fakeLots{lot: l},
		authorizer:   &fakeAuthorizer{allowed: true},
		publisher:    &fakePublisher{},
		tx:           &fakeTxManager{},
	}
}

func (s *scenario) command() voidlot.Command {
	return voidlot.Command{LotID: s.lot.ID(), ExpectedVersion: s.lot.Version(), Reason: "  Registered by mistake  "}
}

func (s *scenario) execute(cmd voidlot.Command) (*lot.Lot, error) {
	ctx := requestcontext.WithActorID(context.Background(), s.organization.ID())
	ctx = requestcontext.WithCorrelationID(ctx, "req-123")
	return voidlot.New(s.actors, s.lots, s.authorizer, s.publisher, s.tx).Execute(ctx, cmd)
}

func (s *scenario) assertNothingPersisted(t *testing.T) {
	t.Helper()
	if len(s.lots.updates) != 0 {
		t.Error("the lot must not be updated")
	}
	for _, e := range s.publisher.events {
		if e.Type == ports.EventLotVoided {
			t.Error("no voided event must be published")
		}
	}
}

func assertFailed(t *testing.T, got *lot.Lot, err error, want exception.Code) {
	t.Helper()
	if !exception.HasCode(err, want) {
		t.Errorf("got error %v, want code %q", err, want)
	}
	if !got.IsDefault() {
		t.Error("expected the default lot on error")
	}
}

func assertInternal(t *testing.T, got *lot.Lot, err error) {
	t.Helper()
	e, ok := exception.From(err)
	if !ok || e.Kind() != exception.KindInternal || !errors.Is(err, errInfra) {
		t.Errorf("expected an internal error wrapping the cause, got %v", err)
	}
	if !got.IsDefault() {
		t.Error("expected the default lot on error")
	}
}

func TestVoidLotSuccess(t *testing.T) {
	s := newScenario(t)
	cmd := s.command()

	got, err := s.execute(cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Run("voids the lot with the trimmed reason and bumps the version", func(t *testing.T) {
		if got != s.lot || got.Status() != lot.StatusVoided || got.VoidReason() != "Registered by mistake" {
			t.Errorf("unexpected lot %+v", got)
		}
		if got.Version() != cmd.ExpectedVersion+1 {
			t.Errorf("version: got %d, want %d", got.Version(), cmd.ExpectedVersion+1)
		}
	})

	t.Run("persists the status inside the transaction guarding the version", func(t *testing.T) {
		if len(s.lots.updates) != 1 || s.tx.calls != 1 {
			t.Fatalf("got %d updates and %d transactions, want 1 and 1", len(s.lots.updates), s.tx.calls)
		}
		u := s.lots.updates[0]
		if u.status != lot.StatusVoided || u.reason != "Registered by mistake" || u.version != 5 || u.expectedVersion != 4 || !u.inTx {
			t.Errorf("unexpected update %+v", u)
		}
	})

	t.Run("asks ABAC to void the lot", func(t *testing.T) {
		a := s.authorizer
		if a.action != ports.ActionVoidLot || a.subject.ActorID != s.organization.ID() || a.subject.Role != actor.TypeOrganization {
			t.Errorf("unexpected action %v or subject %+v", a.action, a.subject)
		}
		if a.resource.Type != ports.ResourceLot || a.resource.ID != s.lot.ID() || a.resource.CustodianID != s.lot.CustodianID() {
			t.Errorf("unexpected resource %+v", a.resource)
		}
	})

	t.Run("publishes LotVoided inside the transaction", func(t *testing.T) {
		if len(s.publisher.events) != 1 || !s.publisher.inTx[0] {
			t.Fatal("expected one event published inside the transaction")
		}
		event := s.publisher.events[0]
		if event.Type != ports.EventLotVoided || event.AggregateType != ports.AggregateLot ||
			event.AggregateID != s.lot.ID() || event.Version != 5 || event.ActorID != s.organization.ID() || event.CorrelationID != "req-123" {
			t.Errorf("unexpected envelope %+v", event)
		}
		payload, ok := event.Payload.(voidlot.LotVoidedPayload)
		if !ok {
			t.Fatalf("unexpected payload type %T", event.Payload)
		}
		if payload.LotID != s.lot.ID() || payload.VoidedBy != s.organization.ID() || payload.Reason != "Registered by mistake" ||
			payload.CustodianID != s.lot.CustodianID() || payload.Stage != "roasting" || payload.Status != "voided" {
			t.Errorf("unexpected payload %+v", payload)
		}
	})
}

func TestVoidLotActorAndLotErrors(t *testing.T) {
	t.Run("the requester does not exist", func(t *testing.T) {
		s := newScenario(t)
		s.actors.actor = actor.Default()

		got, err := s.execute(s.command())
		assertFailed(t, got, err, exception.CodeActorNotFound)
	})

	t.Run("the requester lookup fails", func(t *testing.T) {
		s := newScenario(t)
		s.actors.err = errInfra

		got, err := s.execute(s.command())
		assertInternal(t, got, err)
	})

	t.Run("the lot does not exist", func(t *testing.T) {
		s := newScenario(t)
		s.lots.lot = lot.Default()

		got, err := s.execute(s.command())
		assertFailed(t, got, err, exception.CodeLotNotFound)
	})

	t.Run("the lot lookup fails", func(t *testing.T) {
		s := newScenario(t)
		s.lots.findErr = errInfra

		got, err := s.execute(s.command())
		assertInternal(t, got, err)
	})
}

func TestVoidLotAuthorization(t *testing.T) {
	t.Run("ABAC denies: forbidden, AccessDenied published, nothing persisted", func(t *testing.T) {
		s := newScenario(t)
		s.authorizer.allowed = false

		got, err := s.execute(s.command())
		assertFailed(t, got, err, exception.CodeForbidden)

		if len(s.publisher.events) != 1 || s.publisher.events[0].Type != ports.EventAccessDenied {
			t.Fatal("expected exactly one AccessDenied event")
		}
		payload, ok := s.publisher.events[0].Payload.(access.DeniedPayload)
		if !ok || payload.Action != string(ports.ActionVoidLot) || payload.ResourceID != s.lot.ID().String() {
			t.Errorf("unexpected payload %+v", s.publisher.events[0].Payload)
		}
		s.assertNothingPersisted(t)
		if s.tx.calls != 0 || s.lot.Status() != lot.StatusActive {
			t.Error("expected no transaction and an untouched lot")
		}
	})

	t.Run("ABAC fails", func(t *testing.T) {
		s := newScenario(t)
		s.authorizer.err = errInfra

		got, err := s.execute(s.command())
		assertInternal(t, got, err)
	})
}

func TestVoidLotDomainErrors(t *testing.T) {
	t.Run("the version is stale", func(t *testing.T) {
		s := newScenario(t)
		cmd := s.command()
		cmd.ExpectedVersion = 99

		got, err := s.execute(cmd)
		assertFailed(t, got, err, exception.CodeLotVersionConflict)
		s.assertNothingPersisted(t)
		if s.tx.calls != 0 {
			t.Error("expected no transaction")
		}
	})

	for _, reason := range []string{"", "   "} {
		t.Run("the reason is empty: "+reason, func(t *testing.T) {
			s := newScenario(t)
			cmd := s.command()
			cmd.Reason = reason

			got, err := s.execute(cmd)
			assertFailed(t, got, err, exception.CodeLotEmptyVoidReason)
			s.assertNothingPersisted(t)
		})
	}

	t.Run("the lot is already voided", func(t *testing.T) {
		s := newScenario(t)
		if err := s.lot.Void("First reason"); err != nil {
			t.Fatal(err)
		}
		cmd := s.command()

		got, err := s.execute(cmd)
		assertFailed(t, got, err, exception.CodeLotVoided)
		s.assertNothingPersisted(t)
	})
}

func TestVoidLotPersistenceErrors(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*scenario)
	}{
		{"the transaction fails", func(s *scenario) { s.tx.err = errInfra }},
		{"the status update fails", func(s *scenario) { s.lots.updateErr = errInfra }},
		{"publishing fails", func(s *scenario) {
			s.publisher.err = errInfra
			s.publisher.errFor = ports.EventLotVoided
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
