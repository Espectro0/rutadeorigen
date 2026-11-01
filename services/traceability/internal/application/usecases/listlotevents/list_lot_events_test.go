package listlotevents_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"rutadeorigen/lib/requestcontext"
	"rutadeorigen/traceability/internal/application/output/ports"
	"rutadeorigen/traceability/internal/application/usecases/listlotevents"
	"rutadeorigen/traceability/internal/domain/actor"
	"rutadeorigen/traceability/internal/domain/lot"
	"rutadeorigen/traceability/internal/platform/exception"
	"rutadeorigen/traceability/internal/platform/helper"
)

var errInfra = errors.New("infrastructure failure")

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

type fakeLots struct {
	lot *lot.Lot
	err error
}

func (f fakeLots) ExistsByCode(_ context.Context, _ uuid.UUID, _ string) (bool, error) {
	return false, nil
}

func (f fakeLots) Save(_ context.Context, _ *lot.Lot) error { return nil }

func (f fakeLots) List(_ context.Context, _ lot.Filter, _ lot.Cursor, _ int) ([]*lot.Lot, error) {
	return nil, nil
}

func (f fakeLots) UpdateStage(_ context.Context, _ *lot.Lot, _ int) error { return nil }

func (f fakeLots) UpdateStatus(_ context.Context, _ *lot.Lot, _ int) error { return nil }

func (f fakeLots) UpdateCustodian(_ context.Context, _ *lot.Lot, _ int) error { return nil }

func (f fakeLots) FindByID(_ context.Context, _ uuid.UUID) (*lot.Lot, error) {
	if f.err != nil {
		return lot.Default(), f.err
	}
	return f.lot, nil
}

type fakeEvents struct {
	events []*lot.Event
	err    error
	calls  int
}

func (f *fakeEvents) Save(_ context.Context, _ *lot.Event) error { return nil }

func (f *fakeEvents) FindLast(_ context.Context, _ uuid.UUID) (*lot.Event, error) {
	return lot.DefaultEvent(), nil
}

func (f *fakeEvents) ListByLotID(_ context.Context, _ uuid.UUID) ([]*lot.Event, error) {
	f.calls++
	return f.events, f.err
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
	requester  *actor.Actor
	lot        *lot.Lot
	actors     fakeActors
	lots       fakeLots
	events     *fakeEvents
	authorizer *fakeAuthorizer
}

func newScenario(t *testing.T) *scenario {
	t.Helper()

	requester, err := actor.New("user_01", actor.TypeProducer, "Finca La Esperanza SAS", "user_01@example.com")
	if err != nil {
		t.Fatal(err)
	}
	l := lot.Restore(lot.RestoreParams{
		ID:           uuid.New(),
		FarmID:       uuid.New(),
		CustodianID:  requester.ID(),
		Code:         "LOT-001",
		Variety:      "Castillo",
		Process:      lot.ProcessWashed,
		HarvestDate:  time.Date(2026, time.September, 15, 0, 0, 0, 0, time.UTC),
		CurrentStage: lot.StageFarm,
		Status:       lot.StatusActive,
		Version:      2,
		CreatedAt:    time.Date(2026, time.September, 20, 14, 30, 0, 0, time.UTC),
	})

	first, err := lot.NewTransferEvent(lot.TransferEventParams{
		LotID:         l.ID(),
		Stage:         lot.StageFarm,
		EventDate:     helper.DateUTC(time.Now().AddDate(0, 0, -1)),
		ActorID:       requester.ID(),
		ActorRole:     "producer",
		TargetActorID: uuid.New(),
	})
	if err != nil {
		t.Fatal(err)
	}
	second, err := lot.NewTransferEvent(lot.TransferEventParams{
		LotID:         l.ID(),
		Previous:      first,
		Stage:         lot.StageFarm,
		EventDate:     helper.DateUTC(time.Now().AddDate(0, 0, -1)),
		ActorID:       requester.ID(),
		ActorRole:     "producer",
		TargetActorID: uuid.New(),
	})
	if err != nil {
		t.Fatal(err)
	}

	return &scenario{
		requester:  requester,
		lot:        l,
		actors:     fakeActors{actor: requester},
		lots:       fakeLots{lot: l},
		events:     &fakeEvents{events: []*lot.Event{first, second}},
		authorizer: &fakeAuthorizer{allowed: true},
	}
}

func (s *scenario) execute(id uuid.UUID) (listlotevents.Result, error) {
	ctx := requestcontext.WithActorID(context.Background(), s.requester.ID())
	return listlotevents.New(s.actors, s.lots, s.events, s.authorizer).Execute(ctx, id)
}

func assertEmptyResult(t *testing.T, got listlotevents.Result) {
	t.Helper()
	if got.Events != nil || got.ChainValid {
		t.Errorf("expected the zero result, got %+v", got)
	}
}

func TestListLotEventsReturnsTheValidChain(t *testing.T) {
	s := newScenario(t)

	got, err := s.execute(s.lot.ID())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.Events) != 2 || !got.ChainValid {
		t.Errorf("got %d events and chain valid %v, want 2 and true", len(got.Events), got.ChainValid)
	}
	if got.Events[0].Sequence() != 1 || got.Events[1].Sequence() != 2 {
		t.Error("expected the events in sequence order")
	}
}

func TestListLotEventsAsksABACToReadTheLot(t *testing.T) {
	s := newScenario(t)

	if _, err := s.execute(s.lot.ID()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	a := s.authorizer
	if a.action != ports.ActionReadLot || a.subject.ActorID != s.requester.ID() || a.subject.Role != actor.TypeProducer {
		t.Errorf("unexpected action %v or subject %+v", a.action, a.subject)
	}
	if a.resource.Type != ports.ResourceLot || a.resource.ID != s.lot.ID() || a.resource.CustodianID != s.requester.ID() {
		t.Errorf("unexpected resource %+v", a.resource)
	}
}

func TestListLotEventsReportsATamperedChain(t *testing.T) {
	s := newScenario(t)
	s.events.events = s.events.events[1:]

	got, err := s.execute(s.lot.ID())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.Events) != 1 || got.ChainValid {
		t.Errorf("got %d events and chain valid %v, want 1 and false", len(got.Events), got.ChainValid)
	}
}

func TestListLotEventsOfALotWithoutEvents(t *testing.T) {
	s := newScenario(t)
	s.events.events = nil

	got, err := s.execute(s.lot.ID())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Events == nil || len(got.Events) != 0 || !got.ChainValid {
		t.Errorf("got %+v, want an empty non nil list and a valid chain", got)
	}
}

func TestListLotEventsErrors(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(*scenario)
		wantCode exception.Code
		internal bool
	}{
		{"the requester does not exist", func(s *scenario) { s.actors.actor = actor.Default() }, exception.CodeActorNotFound, false},
		{"the lot does not exist", func(s *scenario) { s.lots.lot = lot.Default() }, exception.CodeLotNotFound, false},
		{"ABAC denies", func(s *scenario) { s.authorizer.allowed = false }, exception.CodeForbidden, false},
		{"the requester lookup fails", func(s *scenario) { s.actors.err = errInfra }, "", true},
		{"the lot lookup fails", func(s *scenario) { s.lots.err = errInfra }, "", true},
		{"ABAC fails", func(s *scenario) { s.authorizer.err = errInfra }, "", true},
		{"the events lookup fails", func(s *scenario) { s.events.err = errInfra }, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newScenario(t)
			tt.setup(s)

			got, err := s.execute(s.lot.ID())

			assertEmptyResult(t, got)
			if tt.internal {
				e, ok := exception.From(err)
				if !ok || e.Kind() != exception.KindInternal || !errors.Is(err, errInfra) {
					t.Errorf("expected an internal error wrapping the cause, got %v", err)
				}
				return
			}
			if !exception.HasCode(err, tt.wantCode) {
				t.Errorf("got %v, want code %q", err, tt.wantCode)
			}
		})
	}
}

func TestListLotEventsDoesNotReadEventsWhenAccessIsDenied(t *testing.T) {
	s := newScenario(t)
	s.authorizer.allowed = false

	_, _ = s.execute(s.lot.ID())

	if s.events.calls != 0 {
		t.Errorf("got %d event reads, want 0", s.events.calls)
	}
}
