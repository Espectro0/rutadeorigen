package projectlotcreated_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"rutadeorigen/qr/internal/application/usecases/projectlotcreated"
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
	deleteErr       error
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
	return c.deleteErr
}

type fixture struct {
	tx        *fakeTx
	views     *fakeViews
	processed *fakeProcessed
	cache     *fakeCache
	uc        *projectlotcreated.ProjectLotCreated
}

func newFixture() *fixture {
	tx := &fakeTx{}
	f := &fixture{
		tx:        tx,
		views:     &fakeViews{tx: tx},
		processed: &fakeProcessed{tx: tx, first: true},
		cache:     &fakeCache{tx: tx},
	}
	f.uc = projectlotcreated.New(f.views, f.processed, f.cache, f.tx)
	return f
}

func validParams() lotview.NewParams {
	return lotview.NewParams{
		LotID:        uuid.New(),
		Code:         "LT-001",
		Variety:      "Castillo",
		Process:      lotview.ProcessWashed,
		HarvestDate:  time.Now().AddDate(0, -1, 0),
		FarmID:       uuid.New(),
		FarmName:     "La Esperanza",
		FarmAltitude: 1800,
		ProducerID:   uuid.New(),
		ProducerName: "Maria Lopez",
		CustodianID:  uuid.New(),
		CurrentStage: lotview.StageNone,
		Status:       lotview.StatusActive,
		Version:      1,
		CreatedAt:    time.Now().Add(-time.Hour),
	}
}

func validCommand() projectlotcreated.Command {
	return projectlotcreated.Command{
		MessageID: uuid.New(),
		EventType: "lot.created",
		Lot:       validParams(),
	}
}

func restoredWithVersion(p lotview.NewParams, version int) *lotview.LotView {
	return lotview.Restore(lotview.RestoreParams{
		LotID:   p.LotID,
		Code:    p.Code,
		Variety: p.Variety,
		Version: version,
	})
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

func TestProjectLotCreatedExecute(t *testing.T) {
	ctx := context.Background()

	t.Run("stores the lot view and registers the message in one transaction", func(t *testing.T) {
		f := newFixture()
		cmd := validCommand()

		err := f.uc.Execute(ctx, cmd)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if f.tx.calls != 1 || f.views.usedOutside || f.processed.usedOutside {
			t.Errorf("expected one transaction around every write, got %d calls", f.tx.calls)
		}
		if len(f.processed.registered) != 1 || f.processed.registered[0] != cmd.MessageID || f.processed.types[0] != "lot.created" {
			t.Errorf("unexpected registered messages: %v %v", f.processed.registered, f.processed.types)
		}
		if len(f.views.saved) != 1 {
			t.Fatalf("expected one saved view, got %d", len(f.views.saved))
		}
		saved := f.views.saved[0]
		if saved.LotID() != cmd.Lot.LotID || saved.Code() != "LT-001" || saved.Version() != 1 {
			t.Errorf("unexpected saved view: %+v", saved)
		}
	})

	t.Run("invalidates the cache after the transaction", func(t *testing.T) {
		f := newFixture()
		cmd := validCommand()

		err := f.uc.Execute(ctx, cmd)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(f.cache.deleted) != 1 || f.cache.deleted[0] != cmd.Lot.LotID {
			t.Errorf("expected the lot to be evicted, got %v", f.cache.deleted)
		}
		if f.cache.deletedInsideTx {
			t.Error("the cache must be invalidated after the commit")
		}
	})

	t.Run("ignores a cache invalidation failure", func(t *testing.T) {
		f := newFixture()
		f.cache.deleteErr = errBackend

		if err := f.uc.Execute(ctx, validCommand()); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("skips a message that was already processed", func(t *testing.T) {
		f := newFixture()
		f.processed.first = false

		err := f.uc.Execute(ctx, validCommand())

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(f.views.saved) != 0 {
			t.Errorf("a duplicate message must not save anything, got %d saves", len(f.views.saved))
		}
	})

	t.Run("replaces an older version", func(t *testing.T) {
		f := newFixture()
		cmd := validCommand()
		cmd.Lot.Version = 3
		f.views.current = restoredWithVersion(cmd.Lot, 2)

		err := f.uc.Execute(ctx, cmd)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(f.views.saved) != 1 || f.views.saved[0].Version() != 3 {
			t.Errorf("expected version 3 to be saved, got %+v", f.views.saved)
		}
	})

	t.Run("keeps a newer or equal version", func(t *testing.T) {
		for _, current := range []int{2, 5} {
			f := newFixture()
			cmd := validCommand()
			cmd.Lot.Version = 2
			f.views.current = restoredWithVersion(cmd.Lot, current)

			err := f.uc.Execute(ctx, cmd)

			if err != nil {
				t.Fatalf("current %d: unexpected error: %v", current, err)
			}
			if len(f.views.saved) != 0 {
				t.Errorf("current %d: an older event must not overwrite the view", current)
			}
		}
	})

	t.Run("rejects an invalid message before opening a transaction", func(t *testing.T) {
		tests := []struct {
			name   string
			mutate func(c *projectlotcreated.Command)
		}{
			{"default message ID", func(c *projectlotcreated.Command) { c.MessageID = uuid.Nil }},
			{"empty event type", func(c *projectlotcreated.Command) { c.EventType = "  " }},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				f := newFixture()
				cmd := validCommand()
				tt.mutate(&cmd)

				err := f.uc.Execute(ctx, cmd)

				assertCode(t, err, exception.CodeEventInvalidPayload)
				if f.tx.calls != 0 {
					t.Errorf("expected no transaction, got %d", f.tx.calls)
				}
			})
		}
	})

	t.Run("rejects an invalid lot before opening a transaction", func(t *testing.T) {
		f := newFixture()
		cmd := validCommand()
		cmd.Lot.Variety = ""

		err := f.uc.Execute(ctx, cmd)

		assertCode(t, err, exception.CodeLotViewInvalidVariety)
		if f.tx.calls != 0 || len(f.cache.deleted) != 0 {
			t.Errorf("expected no transaction or eviction, got %d and %v", f.tx.calls, f.cache.deleted)
		}
	})

	t.Run("wraps storage failures and keeps the cache untouched", func(t *testing.T) {
		tests := []struct {
			name  string
			setup func(f *fixture)
		}{
			{"register failure", func(f *fixture) { f.processed.err = errBackend }},
			{"find failure", func(f *fixture) { f.views.findErr = errBackend }},
			{"save failure", func(f *fixture) { f.views.saveErr = errBackend }},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				f := newFixture()
				tt.setup(f)

				err := f.uc.Execute(ctx, validCommand())

				assertCode(t, err, exception.CodeInternal)
				if !errors.Is(err, errBackend) {
					t.Errorf("the cause must be preserved, got %v", err)
				}
				if len(f.cache.deleted) != 0 {
					t.Errorf("a failed projection must not evict the cache, got %v", f.cache.deleted)
				}
			})
		}
	})
}
