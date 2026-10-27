package registerstage_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"rutadeorigen/lib/requestcontext"
	"rutadeorigen/traceability/internal/application/output/ports"
	"rutadeorigen/traceability/internal/application/usecases/access"
	"rutadeorigen/traceability/internal/application/usecases/registerstage"
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

type updateCall struct {
	stage           lot.Stage
	version         int
	expectedVersion int
	inTx            bool
}

type fakeLots struct {
	byID      map[uuid.UUID]*lot.Lot
	findErr   error
	updateErr error
	updates   []updateCall
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

func (r *fakeLots) FindByID(_ context.Context, id uuid.UUID) (*lot.Lot, error) {
	if r.findErr != nil {
		return lot.Default(), r.findErr
	}
	if l, ok := r.byID[id]; ok {
		return l, nil
	}
	return lot.Default(), nil
}

func (r *fakeLots) UpdateStage(ctx context.Context, l *lot.Lot, expectedVersion int) error {
	if r.updateErr != nil {
		return r.updateErr
	}
	r.updates = append(r.updates, updateCall{
		stage:           l.CurrentStage(),
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

func (r *fakeEvents) ListByLotID(_ context.Context, lotID uuid.UUID) ([]*lot.Event, error) {
	events := make([]*lot.Event, 0, len(r.saved))
	for _, e := range r.saved {
		if e.LotID() == lotID {
			events = append(events, e)
		}
	}
	return events, nil
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
	lot        *lot.Lot
	actors     fakeActors
	lots       *fakeLots
	events     *fakeEvents
	authorizer *fakeAuthorizer
	publisher  *fakePublisher
	tx         *fakeTxManager
}

func newScenario(t *testing.T) *scenario {
	t.Helper()

	producer, err := actor.New("user_01", actor.TypeProducer, "Finca La Esperanza SAS", "user_01@example.com")
	if err != nil {
		t.Fatal(err)
	}
	harvest := helper.DateUTC(time.Now().AddDate(0, -1, 0))
	l, err := lot.New(uuid.New(), producer.ID(), "LT-001", "Castillo", lot.ProcessWashed, harvest)
	if err != nil {
		t.Fatal(err)
	}

	return &scenario{
		producer:   producer,
		lot:        l,
		actors:     fakeActors{byID: map[uuid.UUID]*actor.Actor{producer.ID(): producer}},
		lots:       &fakeLots{byID: map[uuid.UUID]*lot.Lot{l.ID(): l}},
		events:     &fakeEvents{},
		authorizer: &fakeAuthorizer{allowed: true},
		publisher:  &fakePublisher{},
		tx:         &fakeTxManager{},
	}
}

func (s *scenario) useCase() *registerstage.RegisterStage {
	return registerstage.New(s.actors, s.lots, s.events, s.authorizer, s.publisher, s.tx)
}

func (s *scenario) ctx() context.Context {
	ctx := requestcontext.WithActorID(context.Background(), s.producer.ID())
	return requestcontext.WithCorrelationID(ctx, "req-123")
}

func (s *scenario) execute(cmd registerstage.Command) (registerstage.Result, error) {
	return s.useCase().Execute(s.ctx(), cmd)
}

func (s *scenario) farmCommand() registerstage.Command {
	return registerstage.Command{
		LotID:           s.lot.ID(),
		ExpectedVersion: s.lot.Version(),
		Stage:           lot.StageFarm,
		EventDate:       helper.DateUTC(time.Now().AddDate(0, 0, -1)),
		Note:            "  Dried on raised beds  ",
		Details: lot.FarmDetails{
			DryingMethod:      lot.DryingMethodAfricanBed,
			DryingDays:        15,
			MoisturePercent:   11.5,
			ParchmentWeightKg: 120.5,
		},
	}
}

func (s *scenario) roastingCommand() registerstage.Command {
	cmd := s.farmCommand()
	cmd.Stage = lot.StageRoasting
	cmd.Details = lot.RoastingDetails{
		RoastLevel:      lot.RoastLevelMedium,
		DurationMinutes: 14,
		GreenWeightKg:   100,
		RoastedWeightKg: 84,
	}
	return cmd
}

func (s *scenario) packagingCommand() registerstage.Command {
	cmd := s.farmCommand()
	cmd.Stage = lot.StagePackaging
	cmd.Details = lot.PackagingDetails{
		Grind:          lot.GrindWholeBean,
		PackageWeightG: 250,
		Units:          40,
		BestBefore:     time.Now().AddDate(0, 6, 0).Format(time.DateOnly),
	}
	return cmd
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
		if e.Type == ports.EventLotStageRegistered {
			t.Error("no stage event must be published")
		}
	}
}

func assertFailed(t *testing.T, got registerstage.Result, err error, want exception.Code) {
	t.Helper()
	if !exception.HasCode(err, want) {
		t.Errorf("got error %v, want code %q", err, want)
	}
	assertDefaultResult(t, got)
}

func assertInternal(t *testing.T, got registerstage.Result, err error) {
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

func assertDefaultResult(t *testing.T, got registerstage.Result) {
	t.Helper()
	if !got.Event.IsDefault() {
		t.Error("expected the default event on error")
	}
	if got.LotVersion != 0 {
		t.Errorf("got version %d, want 0 on error", got.LotVersion)
	}
}

func TestRegisterStageSuccess(t *testing.T) {
	s := newScenario(t)
	cmd := s.farmCommand()

	got, err := s.execute(cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Run("returns the registered event", func(t *testing.T) {
		if got.Event.IsDefault() {
			t.Fatal("expected a real event")
		}
		if got.Event.LotID() != s.lot.ID() || got.Event.Stage() != lot.StageFarm || got.Event.Type() != lot.EventTypeStage {
			t.Error("unexpected event data")
		}
		if got.Event.Sequence() != 1 || got.Event.PreviousHash() != lot.GenesisHash || !got.Event.VerifyHash() {
			t.Error("expected the first event of a valid chain")
		}
		if got.Event.Note() != "Dried on raised beds" {
			t.Errorf("got note %q, want it trimmed", got.Event.Note())
		}
		if got.Event.ActorID() != s.producer.ID() || got.Event.ActorRole() != actor.TypeProducer.String() {
			t.Error("unexpected actor in the lot event")
		}
	})

	t.Run("returns the new lot version", func(t *testing.T) {
		if got.LotVersion != 2 {
			t.Errorf("got version %d, want 2", got.LotVersion)
		}
	})

	t.Run("advances the lot", func(t *testing.T) {
		if s.lot.CurrentStage() != lot.StageFarm || s.lot.Version() != 2 {
			t.Errorf("unexpected lot state: stage %q, version %d", s.lot.CurrentStage(), s.lot.Version())
		}
	})

	t.Run("updates the lot guarded by the version the client sent", func(t *testing.T) {
		if len(s.lots.updates) != 1 {
			t.Fatalf("got %d updates, want 1", len(s.lots.updates))
		}
		u := s.lots.updates[0]
		if u.stage != lot.StageFarm || u.version != 2 || u.expectedVersion != 1 {
			t.Errorf("unexpected update: %+v", u)
		}
	})

	t.Run("saves the event", func(t *testing.T) {
		if len(s.events.saved) != 1 || s.events.saved[0] != got.Event {
			t.Error("expected the returned event to be saved once")
		}
	})

	t.Run("asks ABAC with the lot as resource", func(t *testing.T) {
		a := s.authorizer
		if a.action != ports.ActionRegisterStage {
			t.Errorf("action: got %v, want %v", a.action, ports.ActionRegisterStage)
		}
		if a.subject.ActorID != s.producer.ID() || a.subject.Role != actor.TypeProducer {
			t.Error("unexpected subject")
		}
		if a.resource.Type != ports.ResourceLot || a.resource.ID != s.lot.ID() || a.resource.CustodianID != s.lot.CustodianID() {
			t.Error("unexpected resource")
		}
	})

	t.Run("publishes StageRegistered with the new lot version", func(t *testing.T) {
		if len(s.publisher.events) != 1 {
			t.Fatalf("got %d events, want 1", len(s.publisher.events))
		}
		e := s.publisher.events[0]
		if e.Type != ports.EventLotStageRegistered || e.AggregateType != ports.AggregateLot {
			t.Errorf("unexpected event type %v / %v", e.Type, e.AggregateType)
		}
		if e.AggregateID != s.lot.ID() || e.Version != 2 {
			t.Errorf("unexpected aggregate %s / version %d", e.AggregateID, e.Version)
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
	})

	t.Run("the payload describes the new state", func(t *testing.T) {
		payload, ok := s.publisher.events[0].Payload.(registerstage.StageRegisteredPayload)
		if !ok {
			t.Fatalf("unexpected payload type %T", s.publisher.events[0].Payload)
		}
		want := registerstage.StageRegisteredPayload{
			LotID:     s.lot.ID(),
			Stage:     "farm",
			Status:    "active",
			EventID:   got.Event.ID(),
			Sequence:  1,
			EventDate: got.Event.EventDate().Format(time.DateOnly),
		}
		if payload != want {
			t.Errorf("got %+v, want %+v", payload, want)
		}
	})

	t.Run("everything happens in one transaction", func(t *testing.T) {
		if s.tx.calls != 1 {
			t.Errorf("got %d transactions, want 1", s.tx.calls)
		}
		if !s.lots.updates[0].inTx || !s.events.savedTx[0] || !s.publisher.inTx[0] {
			t.Error("expected update, save and publish inside the transaction")
		}
	})
}

func TestRegisterStageFullPath(t *testing.T) {
	s := newScenario(t)

	farm, err := s.execute(s.farmCommand())
	if err != nil {
		t.Fatalf("farm: unexpected error: %v", err)
	}
	roasting, err := s.execute(s.roastingCommand())
	if err != nil {
		t.Fatalf("roasting: unexpected error: %v", err)
	}
	packaging, err := s.execute(s.packagingCommand())
	if err != nil {
		t.Fatalf("packaging: unexpected error: %v", err)
	}

	if farm.LotVersion != 2 || roasting.LotVersion != 3 || packaging.LotVersion != 4 {
		t.Errorf("unexpected versions: %d, %d, %d", farm.LotVersion, roasting.LotVersion, packaging.LotVersion)
	}
	if s.lot.CurrentStage() != lot.StagePackaging {
		t.Errorf("got stage %q, want packaging", s.lot.CurrentStage())
	}
	if roasting.Event.Sequence() != 2 || roasting.Event.PreviousHash() != farm.Event.Hash() {
		t.Error("expected the roasting event to follow the farm event")
	}
	if packaging.Event.Sequence() != 3 || packaging.Event.PreviousHash() != roasting.Event.Hash() {
		t.Error("expected the packaging event to follow the roasting event")
	}
	for i, e := range s.events.saved {
		if !e.VerifyHash() {
			t.Errorf("event %d does not verify", i+1)
		}
	}

	versions := []int{2, 3, 4}
	for i, e := range s.publisher.events {
		if e.Type != ports.EventLotStageRegistered || e.Version != versions[i] {
			t.Errorf("event %d: got %v version %d, want version %d", i+1, e.Type, e.Version, versions[i])
		}
	}
}

func TestRegisterStageOutOfOrder(t *testing.T) {
	tests := []struct {
		name    string
		prepare func(*scenario)
		command func(*scenario) registerstage.Command
	}{
		{"roasting before farm", func(*scenario) {}, (*scenario).roastingCommand},
		{"packaging before farm", func(*scenario) {}, (*scenario).packagingCommand},
		{"packaging skipping roasting", func(s *scenario) { advance(t, s.lot, lot.StageFarm) }, (*scenario).packagingCommand},
		{"farm twice", func(s *scenario) { advance(t, s.lot, lot.StageFarm) }, (*scenario).farmCommand},
		{"farm after roasting", func(s *scenario) { advance(t, s.lot, lot.StageFarm, lot.StageRoasting) }, (*scenario).farmCommand},
		{"roasting after packaging", func(s *scenario) {
			advance(t, s.lot, lot.StageFarm, lot.StageRoasting, lot.StagePackaging)
		}, (*scenario).roastingCommand},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newScenario(t)
			tt.prepare(s)

			got, err := s.execute(tt.command(s))
			assertFailed(t, got, err, exception.CodeLotStageOutOfOrder)
			s.assertNothingPersisted(t)
		})
	}
}

func advance(t *testing.T, l *lot.Lot, stages ...lot.Stage) {
	t.Helper()
	for _, stage := range stages {
		if err := l.AdvanceTo(stage); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}
}

func TestRegisterStageVersionConflict(t *testing.T) {
	t.Run("stale version", func(t *testing.T) {
		for _, expected := range []int{0, 5} {
			s := newScenario(t)
			cmd := s.farmCommand()
			cmd.ExpectedVersion = expected

			got, err := s.execute(cmd)
			assertFailed(t, got, err, exception.CodeLotVersionConflict)
			s.assertNothingPersisted(t)
			if s.tx.calls != 0 {
				t.Errorf("version %d: got %d transactions, want 0", expected, s.tx.calls)
			}
			if s.lot.Version() != 1 || s.lot.CurrentStage() != lot.StageNone {
				t.Errorf("version %d: the lot must not change", expected)
			}
		}
	})

	t.Run("stale version after another registration", func(t *testing.T) {
		s := newScenario(t)
		stale := s.farmCommand()
		if _, err := s.execute(stale); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		roasting := s.roastingCommand()
		roasting.ExpectedVersion = stale.ExpectedVersion

		got, err := s.execute(roasting)
		assertFailed(t, got, err, exception.CodeLotVersionConflict)
		if len(s.events.saved) != 1 {
			t.Errorf("got %d saved events, want 1", len(s.events.saved))
		}
	})

	t.Run("the repository detects a concurrent change", func(t *testing.T) {
		s := newScenario(t)
		s.lots.updateErr = exception.New(exception.CodeLotVersionConflict)

		got, err := s.execute(s.farmCommand())
		assertFailed(t, got, err, exception.CodeLotVersionConflict)
		s.assertNothingPersisted(t)
	})
}

func TestRegisterStageVoidedLot(t *testing.T) {
	s := newScenario(t)
	if err := s.lot.Void("Registered by mistake"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, err := s.execute(s.farmCommand())
	assertFailed(t, got, err, exception.CodeLotVoided)
	s.assertNothingPersisted(t)
}

func TestRegisterStageLotErrors(t *testing.T) {
	t.Run("lot not found", func(t *testing.T) {
		s := newScenario(t)
		cmd := s.farmCommand()
		cmd.LotID = uuid.New()

		got, err := s.execute(cmd)
		assertFailed(t, got, err, exception.CodeLotNotFound)
		s.assertNothingPersisted(t)
	})

	t.Run("lot repository fails", func(t *testing.T) {
		s := newScenario(t)
		s.lots.findErr = errInfra

		got, err := s.execute(s.farmCommand())
		assertInternal(t, got, err)
	})
}

func TestRegisterStageActorErrors(t *testing.T) {
	t.Run("no actor in the context", func(t *testing.T) {
		s := newScenario(t)

		got, err := s.useCase().Execute(context.Background(), s.farmCommand())
		assertFailed(t, got, err, exception.CodeActorNotFound)
		s.assertNothingPersisted(t)
	})

	t.Run("inactive actor", func(t *testing.T) {
		s := newScenario(t)
		s.actors.byID[s.producer.ID()] = actor.Restore(s.producer.ID(), "user_01", actor.TypeProducer,
			"Finca La Esperanza SAS", "user_01@example.com", actor.StatusInactive, time.Now())

		got, err := s.execute(s.farmCommand())
		assertFailed(t, got, err, exception.CodeActorNotFound)
		s.assertNothingPersisted(t)
	})

	t.Run("actor repository fails", func(t *testing.T) {
		s := newScenario(t)
		s.actors.err = errInfra

		got, err := s.execute(s.farmCommand())
		assertInternal(t, got, err)
	})
}

func TestRegisterStageAuthorization(t *testing.T) {
	t.Run("ABAC denies: forbidden, AccessDenied published, nothing persisted", func(t *testing.T) {
		s := newScenario(t)
		s.authorizer.allowed = false

		got, err := s.execute(s.farmCommand())
		assertFailed(t, got, err, exception.CodeForbidden)

		e, _ := exception.From(err)
		if e.Kind() != exception.KindForbidden {
			t.Errorf("Kind: got %v, want %v", e.Kind(), exception.KindForbidden)
		}
		if len(s.publisher.events) != 1 || s.publisher.events[0].Type != ports.EventAccessDenied {
			t.Fatal("expected exactly one AccessDenied event")
		}
		s.assertNothingPersisted(t)
		if s.tx.calls != 0 || s.publisher.inTx[0] {
			t.Error("expected AccessDenied outside a transaction")
		}
	})

	t.Run("AccessDenied describes the attempt", func(t *testing.T) {
		s := newScenario(t)
		s.authorizer.allowed = false

		_, _ = s.execute(s.farmCommand())

		payload, ok := s.publisher.events[0].Payload.(access.DeniedPayload)
		if !ok {
			t.Fatalf("unexpected payload type %T", s.publisher.events[0].Payload)
		}
		want := access.DeniedPayload{
			Action:       string(ports.ActionRegisterStage),
			ResourceType: string(ports.ResourceLot),
			ResourceID:   s.lot.ID().String(),
		}
		if payload != want {
			t.Errorf("got %+v, want %+v", payload, want)
		}
		if s.publisher.events[0].ActorID != s.producer.ID() {
			t.Error("expected the requester as the event actor")
		}
	})

	t.Run("denied before revealing a stale version", func(t *testing.T) {
		s := newScenario(t)
		s.authorizer.allowed = false
		cmd := s.farmCommand()
		cmd.ExpectedVersion = 99

		got, err := s.execute(cmd)
		assertFailed(t, got, err, exception.CodeForbidden)
	})

	t.Run("ABAC fails", func(t *testing.T) {
		s := newScenario(t)
		s.authorizer.err = errInfra

		got, err := s.execute(s.farmCommand())
		assertInternal(t, got, err)
	})

	t.Run("publishing AccessDenied fails", func(t *testing.T) {
		s := newScenario(t)
		s.authorizer.allowed = false
		s.publisher.err = errInfra

		got, err := s.execute(s.farmCommand())
		assertInternal(t, got, err)
	})
}

func TestRegisterStageDomainErrors(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*registerstage.Command)
		want   exception.Code
	}{
		{"nil details", func(c *registerstage.Command) { c.Details = nil }, exception.CodeStageInvalidDetails},
		{"details from another stage", func(c *registerstage.Command) {
			c.Details = lot.RoastingDetails{RoastLevel: lot.RoastLevelDark, DurationMinutes: 10, GreenWeightKg: 10, RoastedWeightKg: 8}
		}, exception.CodeStageInvalidDetails},
		{"invalid details", func(c *registerstage.Command) { c.Details = lot.FarmDetails{} }, exception.CodeStageInvalidDryingMethod},
		{"future event date", func(c *registerstage.Command) { c.EventDate = time.Now().AddDate(0, 0, 2) }, exception.CodeLotInvalidEventDate},
		{"event date before harvest", func(c *registerstage.Command) { c.EventDate = time.Now().AddDate(0, -2, 0) }, exception.CodeLotInvalidEventDate},
		{"default event date", func(c *registerstage.Command) { c.EventDate = time.Time{} }, exception.CodeLotInvalidEventDate},
		{"note too long", func(c *registerstage.Command) {
			note := make([]rune, 501)
			for i := range note {
				note[i] = 'a'
			}
			c.Note = string(note)
		}, exception.CodeLotInvalidNote},
		{"invalid stage", func(c *registerstage.Command) { c.Stage = lot.StageUnknown }, exception.CodeLotStageOutOfOrder},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newScenario(t)
			cmd := s.farmCommand()
			tt.mutate(&cmd)

			got, err := s.execute(cmd)
			assertFailed(t, got, err, tt.want)
			s.assertNothingPersisted(t)
		})
	}
}

func TestRegisterStagePersistenceErrors(t *testing.T) {
	tests := []struct {
		name    string
		prepare func(*scenario)
	}{
		{"reading the last event fails", func(s *scenario) { s.events.findErr = errInfra }},
		{"updating the lot fails", func(s *scenario) { s.lots.updateErr = errInfra }},
		{"saving the event fails", func(s *scenario) { s.events.saveErr = errInfra }},
		{"publishing StageRegistered fails", func(s *scenario) {
			s.publisher.err = errInfra
			s.publisher.errFor = ports.EventLotStageRegistered
		}},
		{"the transaction fails", func(s *scenario) { s.tx.err = errInfra }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newScenario(t)
			tt.prepare(s)

			got, err := s.execute(s.farmCommand())
			assertInternal(t, got, err)
		})
	}

	t.Run("a failure before saving the event leaves no event", func(t *testing.T) {
		s := newScenario(t)
		s.lots.updateErr = errInfra

		_, _ = s.execute(s.farmCommand())
		if len(s.events.saved) != 0 {
			t.Error("no lot event must be saved when updating the lot fails")
		}
	})
}

func TestRegisterStageDoesNotOpenTransactionOnEarlyFailures(t *testing.T) {
	s := newScenario(t)
	cmd := s.farmCommand()
	cmd.ExpectedVersion = 7

	_, _ = s.execute(cmd)
	if s.tx.calls != 0 {
		t.Errorf("got %d transactions, want 0", s.tx.calls)
	}
}

func (r fakeActors) FindByUserID(_ context.Context, _ string) (*actor.Actor, error) {
	return actor.Default(), nil
}

func (r fakeActors) Save(_ context.Context, _ *actor.Actor) error {
	return nil
}
