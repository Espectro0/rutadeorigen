package deletefarm_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"rutadeorigen/lib/requestcontext"
	"rutadeorigen/traceability/internal/application/output/ports"
	"rutadeorigen/traceability/internal/application/usecases/access"
	"rutadeorigen/traceability/internal/application/usecases/deletefarm"
	"rutadeorigen/traceability/internal/domain/actor"
	"rutadeorigen/traceability/internal/domain/farm"
	"rutadeorigen/traceability/internal/domain/lot"
	"rutadeorigen/traceability/internal/platform/exception"
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

type fakeFarms struct {
	farm      *farm.Farm
	findErr   error
	deleteErr error
	deleted   []uuid.UUID
}

func (f *fakeFarms) FindByID(_ context.Context, _ uuid.UUID) (*farm.Farm, error) {
	if f.findErr != nil {
		return farm.Default(), f.findErr
	}
	return f.farm, nil
}

func (f *fakeFarms) Save(_ context.Context, _ *farm.Farm) error { return nil }

func (f *fakeFarms) List(_ context.Context) ([]*farm.Farm, error) { return nil, nil }

func (f *fakeFarms) Delete(_ context.Context, id uuid.UUID) error {
	if f.deleteErr != nil {
		return f.deleteErr
	}
	f.deleted = append(f.deleted, id)
	return nil
}

type fakeLots struct {
	rows    []*lot.Lot
	listErr error
	filter  lot.Filter
	limit   int
	calls   int
}

func (f *fakeLots) ExistsByCode(_ context.Context, _ uuid.UUID, _ string) (bool, error) {
	return false, nil
}

func (f *fakeLots) Save(_ context.Context, _ *lot.Lot) error { return nil }

func (f *fakeLots) FindByID(_ context.Context, _ uuid.UUID) (*lot.Lot, error) {
	return lot.Default(), nil
}

func (f *fakeLots) UpdateStage(_ context.Context, _ *lot.Lot, _ int) error { return nil }

func (f *fakeLots) UpdateCustodian(_ context.Context, _ *lot.Lot, _ int) error { return nil }

func (f *fakeLots) UpdateStatus(_ context.Context, _ *lot.Lot, _ int) error { return nil }

func (f *fakeLots) List(_ context.Context, filter lot.Filter, _ lot.Cursor, limit int) ([]*lot.Lot, error) {
	f.calls++
	f.filter, f.limit = filter, limit
	return f.rows, f.listErr
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
	farms      *fakeFarms
	lots       *fakeLots
	authorizer *fakeAuthorizer
	publisher  *fakePublisher
}

func newScenario(t *testing.T) *scenario {
	t.Helper()

	producer, err := actor.New("user_01", actor.TypeProducer, "Finca La Esperanza SAS", "user_01@example.com")
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
		actors:     fakeActors{actor: producer},
		farms:      &fakeFarms{farm: f},
		lots:       &fakeLots{},
		authorizer: &fakeAuthorizer{allowed: true},
		publisher:  &fakePublisher{},
	}
}

func (s *scenario) execute(id uuid.UUID) error {
	ctx := requestcontext.WithActorID(context.Background(), s.producer.ID())
	return deletefarm.New(s.actors, s.farms, s.lots, s.authorizer, s.publisher).Execute(ctx, id)
}

func (s *scenario) assertNotDeleted(t *testing.T) {
	t.Helper()
	if len(s.farms.deleted) != 0 {
		t.Error("the farm must not be deleted")
	}
}

func assertInternal(t *testing.T, err error) {
	t.Helper()
	e, ok := exception.From(err)
	if !ok || e.Kind() != exception.KindInternal || !errors.Is(err, errInfra) {
		t.Errorf("expected an internal error wrapping the cause, got %v", err)
	}
}

func TestDeleteFarmSuccess(t *testing.T) {
	s := newScenario(t)

	if err := s.execute(s.farm.ID()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(s.farms.deleted) != 1 || s.farms.deleted[0] != s.farm.ID() {
		t.Errorf("got %v, want the farm deleted once", s.farms.deleted)
	}
	if s.lots.calls != 1 || s.lots.filter.FarmID != s.farm.ID() || s.lots.limit != 1 {
		t.Errorf("expected a single lookup of one lot of the farm, got %d calls, filter %+v, limit %d", s.lots.calls, s.lots.filter, s.lots.limit)
	}

	a := s.authorizer
	if a.action != ports.ActionDeleteFarm || a.subject.ActorID != s.producer.ID() || a.subject.Role != actor.TypeProducer {
		t.Errorf("unexpected action %v or subject %+v", a.action, a.subject)
	}
	if a.resource != (ports.Resource{Type: ports.ResourceFarm, ID: s.farm.ID(), OwnerID: s.producer.ID()}) {
		t.Errorf("unexpected resource %+v", a.resource)
	}
	if len(s.publisher.events) != 0 {
		t.Error("no event is published for a successful delete")
	}
}

func TestDeleteFarmErrors(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(*scenario)
		wantCode exception.Code
		internal bool
	}{
		{"the requester does not exist", func(s *scenario) { s.actors.actor = actor.Default() }, exception.CodeActorNotFound, false},
		{"the farm does not exist", func(s *scenario) { s.farms.farm = farm.Default() }, exception.CodeFarmNotFound, false},
		{"the farm has lots", func(s *scenario) {
			s.lots.rows = []*lot.Lot{lot.Restore(lot.RestoreParams{ID: uuid.New(), FarmID: s.farm.ID(), CustodianID: uuid.New(), Code: "LOT-001",
				Variety: "Castillo", Process: lot.ProcessWashed, HarvestDate: time.Now().UTC(), CurrentStage: lot.StageNone,
				Status: lot.StatusActive, Version: 1, CreatedAt: time.Now().UTC()})}
		}, exception.CodeFarmHasLots, false},
		{"the repository reports lots on delete", func(s *scenario) { s.farms.deleteErr = exception.New(exception.CodeFarmHasLots) }, exception.CodeFarmHasLots, false},
		{"the requester lookup fails", func(s *scenario) { s.actors.err = errInfra }, "", true},
		{"the farm lookup fails", func(s *scenario) { s.farms.findErr = errInfra }, "", true},
		{"ABAC fails", func(s *scenario) { s.authorizer.err = errInfra }, "", true},
		{"the lots lookup fails", func(s *scenario) { s.lots.listErr = errInfra }, "", true},
		{"the delete fails", func(s *scenario) { s.farms.deleteErr = errInfra }, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newScenario(t)
			tt.setup(s)

			err := s.execute(s.farm.ID())

			s.assertNotDeleted(t)
			if tt.internal {
				assertInternal(t, err)
				return
			}
			if !exception.HasCode(err, tt.wantCode) {
				t.Errorf("got %v, want code %q", err, tt.wantCode)
			}
		})
	}
}

func TestDeleteFarmDeniedPublishesAccessDenied(t *testing.T) {
	s := newScenario(t)
	s.authorizer.allowed = false

	err := s.execute(s.farm.ID())

	if !exception.HasCode(err, exception.CodeForbidden) {
		t.Errorf("got %v, want code %q", err, exception.CodeForbidden)
	}
	if len(s.publisher.events) != 1 || s.publisher.events[0].Type != ports.EventAccessDenied {
		t.Fatal("expected exactly one AccessDenied event")
	}
	payload, ok := s.publisher.events[0].Payload.(access.DeniedPayload)
	if !ok || payload.Action != string(ports.ActionDeleteFarm) || payload.ResourceID != s.farm.ID().String() {
		t.Errorf("unexpected payload %+v", s.publisher.events[0].Payload)
	}
	s.assertNotDeleted(t)
	if s.lots.calls != 0 {
		t.Error("the lots must not be read when access is denied")
	}
}
