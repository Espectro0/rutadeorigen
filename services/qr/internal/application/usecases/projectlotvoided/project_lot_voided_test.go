package projectlotvoided_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"rutadeorigen/qr/internal/application/usecases/projectlotvoided"
	"rutadeorigen/qr/internal/domain/lotview"
	"rutadeorigen/qr/internal/platform/exception"
)

var errBackend = errors.New("backend is down")

type fakeTx struct {
	active bool
	calls  int
}

func (t *fakeTx) WithinTx(ctx context.Context, fn func(ctx context.Context) error) error {
	t.calls++
	t.active = true
	defer func() { t.active = false }()
	return fn(ctx)
}

type fakeViews struct {
	tx          *fakeTx
	current     *lotview.LotView
	findErr     error
	saveErr     error
	saved       []*lotview.LotView
	usedOutside bool
}

func (v *fakeViews) FindByID(_ context.Context, _ uuid.UUID) (*lotview.LotView, error) {
	v.usedOutside = v.usedOutside || !v.tx.active
	if v.findErr != nil {
		return lotview.Default(), v.findErr
	}
	if v.current == nil {
		return lotview.Default(), nil
	}
	return v.current, nil
}

func (v *fakeViews) Save(_ context.Context, view *lotview.LotView) error {
	v.usedOutside = v.usedOutside || !v.tx.active
	if v.saveErr != nil {
		return v.saveErr
	}
	v.saved = append(v.saved, view)
	return nil
}

type fakeProcessed struct {
	tx          *fakeTx
	first       bool
	err         error
	registered  []uuid.UUID
	types       []string
	usedOutside bool
}

func (p *fakeProcessed) Register(_ context.Context, id uuid.UUID, eventType string) (bool, error) {
	p.usedOutside = p.usedOutside || !p.tx.active
	if p.err != nil {
		return false, p.err
	}
	p.registered = append(p.registered, id)
	p.types = append(p.types, eventType)
	return p.first, nil
}

type fakeCache struct {
	tx              *fakeTx
	deleted         []uuid.UUID
	deletedInsideTx bool
}

func (c *fakeCache) Get(_ context.Context, _ uuid.UUID) (*lotview.LotView, error) {
	return lotview.Default(), nil
}

func (c *fakeCache) Set(_ context.Context, _ *lotview.LotView) error { return nil }

func (c *fakeCache) Delete(_ context.Context, id uuid.UUID) error {
	c.deletedInsideTx = c.deletedInsideTx || c.tx.active
	c.deleted = append(c.deleted, id)
	return nil
}

func existingView(t *testing.T, lotID uuid.UUID, status lotview.Status, version int) *lotview.LotView {
	t.Helper()

	view, err := lotview.New(lotview.NewParams{
		LotID:        lotID,
		Code:         "LT-001",
		Variety:      "Castillo",
		Process:      lotview.ProcessWashed,
		HarvestDate:  time.Now().AddDate(0, -2, 0),
		FarmID:       uuid.New(),
		FarmName:     "La Esperanza",
		FarmAltitude: 1800,
		ProducerID:   uuid.New(),
		ProducerName: "Maria Lopez",
		CustodianID:  uuid.New(),
		CurrentStage: lotview.StageRoasting,
		Status:       status,
		Version:      version,
		CreatedAt:    time.Now().Add(-time.Hour),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return view
}

func assertCode(t *testing.T, err error, want exception.Code) {
	t.Helper()

	ex, ok := exception.From(err)
	if !ok {
		t.Fatalf("expected an *Exception, got %v", err)
	}
	if ex.Code() != want {
		t.Errorf("Code: got %v, want %v", ex.Code(), want)
	}
}

type fixture struct {
	tx        *fakeTx
	views     *fakeViews
	processed *fakeProcessed
	cache     *fakeCache
	uc        *projectlotvoided.ProjectLotVoided
}

func newFixture() *fixture {
	tx := &fakeTx{}
	f := &fixture{
		tx:        tx,
		views:     &fakeViews{tx: tx},
		processed: &fakeProcessed{tx: tx, first: true},
		cache:     &fakeCache{tx: tx},
	}
	f.uc = projectlotvoided.New(f.views, f.processed, f.cache, f.tx)
	return f
}

func validCommand() projectlotvoided.Command {
	return projectlotvoided.Command{
		MessageID:  uuid.New(),
		EventType:  "lot.voided",
		LotID:      uuid.New(),
		Reason:     "Lote contaminado",
		Version:    3,
		OccurredAt: time.Date(2026, 10, 2, 9, 0, 0, 0, time.UTC),
	}
}

func TestExecuteAppliesTheEventInOneTransaction(t *testing.T) {
	cmd := validCommand()
	f := newFixture()
	f.views.current = existingView(t, cmd.LotID, lotview.StatusActive, 2)

	if err := f.uc.Execute(context.Background(), cmd); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if f.tx.calls != 1 || f.views.usedOutside || f.processed.usedOutside {
		t.Errorf("expected one transaction around every access, got %d calls", f.tx.calls)
	}
	if len(f.processed.registered) != 1 || f.processed.registered[0] != cmd.MessageID || f.processed.types[0] != "lot.voided" {
		t.Errorf("unexpected registered messages: %v %v", f.processed.registered, f.processed.types)
	}
	if len(f.views.saved) != 1 {
		t.Fatalf("expected one saved view, got %d", len(f.views.saved))
	}
	saved := f.views.saved[0]
	if saved.Version() != 3 || !saved.UpdatedAt().Equal(cmd.OccurredAt) {
		t.Errorf("unexpected saved view version %d updated at %v", saved.Version(), saved.UpdatedAt())
	}
	if saved.Status() != lotview.StatusVoided || saved.VoidReason() != "Lote contaminado" {
		t.Errorf("unexpected status %s or reason %q", saved.Status(), saved.VoidReason())
	}
	if len(f.cache.deleted) != 1 || f.cache.deleted[0] != cmd.LotID || f.cache.deletedInsideTx {
		t.Errorf("expected the cache entry to be deleted after the transaction, got %v", f.cache.deleted)
	}
}

func TestExecuteSkipsDuplicatesStaleEventsAndVoidedLots(t *testing.T) {
	tests := []struct {
		name  string
		setup func(f *fixture, cmd projectlotvoided.Command)
	}{
		{"message already processed", func(f *fixture, cmd projectlotvoided.Command) {
			f.processed.first = false
			f.views.current = existingView(t, cmd.LotID, lotview.StatusActive, 2)
		}},
		{"same version", func(f *fixture, cmd projectlotvoided.Command) {
			f.views.current = existingView(t, cmd.LotID, lotview.StatusActive, 3)
		}},
		{"older version", func(f *fixture, cmd projectlotvoided.Command) {
			f.views.current = existingView(t, cmd.LotID, lotview.StatusActive, 7)
		}},
		{"voided lot", func(f *fixture, cmd projectlotvoided.Command) {
			f.views.current = existingView(t, cmd.LotID, lotview.StatusVoided, 2)
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := validCommand()
			f := newFixture()
			tt.setup(f, cmd)

			if err := f.uc.Execute(context.Background(), cmd); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(f.views.saved) != 0 {
				t.Errorf("nothing must be saved, got %d saves", len(f.views.saved))
			}
		})
	}
}

func TestExecuteRejectsAnIncompleteCommand(t *testing.T) {
	for name, mutate := range map[string]func(*projectlotvoided.Command){
		"no message id": func(c *projectlotvoided.Command) { c.MessageID = uuid.Nil },
		"no event type": func(c *projectlotvoided.Command) { c.EventType = " " },
		"no lot id":     func(c *projectlotvoided.Command) { c.LotID = uuid.Nil },
	} {
		t.Run(name, func(t *testing.T) {
			cmd := validCommand()
			mutate(&cmd)
			f := newFixture()

			assertCode(t, f.uc.Execute(context.Background(), cmd), exception.CodeEventInvalidPayload)
			if f.tx.calls != 0 {
				t.Errorf("no transaction expected, got %d", f.tx.calls)
			}
		})
	}
}

func TestExecuteFailsWhenTheLotIsNotProjectedYet(t *testing.T) {
	f := newFixture()

	assertCode(t, f.uc.Execute(context.Background(), validCommand()), exception.CodeLotViewNotFound)
	if len(f.cache.deleted) != 0 {
		t.Error("the cache must not be touched when the transaction fails")
	}
}

func TestExecuteWrapsRepositoryFailures(t *testing.T) {
	for name, setup := range map[string]func(f *fixture, cmd projectlotvoided.Command){
		"processed event": func(f *fixture, _ projectlotvoided.Command) { f.processed.err = errBackend },
		"find":            func(f *fixture, _ projectlotvoided.Command) { f.views.findErr = errBackend },
		"save": func(f *fixture, cmd projectlotvoided.Command) {
			f.views.current = existingView(t, cmd.LotID, lotview.StatusActive, 2)
			f.views.saveErr = errBackend
		},
	} {
		t.Run(name, func(t *testing.T) {
			cmd := validCommand()
			f := newFixture()
			setup(f, cmd)

			err := f.uc.Execute(context.Background(), cmd)

			if !errors.Is(err, errBackend) {
				t.Errorf("expected the backend error, got %v", err)
			}
		})
	}
}
