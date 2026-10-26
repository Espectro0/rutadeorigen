package projectlotstage_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"rutadeorigen/qr/internal/application/usecases/projectlotstage"
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
	found       []uuid.UUID
	saved       []*lotview.LotView
	usedOutside bool
}

func (v *fakeViews) FindByID(_ context.Context, id uuid.UUID) (*lotview.LotView, error) {
	v.usedOutside = v.usedOutside || !v.tx.active
	v.found = append(v.found, id)
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
	uc        *projectlotstage.ProjectLotStage
}

func newFixture() *fixture {
	tx := &fakeTx{}
	f := &fixture{
		tx:        tx,
		views:     &fakeViews{tx: tx},
		processed: &fakeProcessed{tx: tx, first: true},
		cache:     &fakeCache{tx: tx},
	}
	f.uc = projectlotstage.New(f.views, f.processed, f.cache, f.tx)
	return f
}

func existingView(t *testing.T, lotID uuid.UUID, stage lotview.Stage, status lotview.Status, version int) *lotview.LotView {
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
		CurrentStage: stage,
		Status:       status,
		Version:      version,
		CreatedAt:    time.Now().Add(-time.Hour),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return view
}

func validCommand() projectlotstage.Command {
	return projectlotstage.Command{
		MessageID:  uuid.New(),
		EventType:  "lot.stage_registered",
		LotID:      uuid.New(),
		Stage:      lotview.StageRoasting,
		Version:    3,
		OccurredAt: time.Date(2026, 9, 20, 14, 30, 0, 0, time.UTC),
	}
}

func fixtureWithView(t *testing.T, cmd projectlotstage.Command, stage lotview.Stage, status lotview.Status, version int) *fixture {
	t.Helper()

	f := newFixture()
	f.views.current = existingView(t, cmd.LotID, stage, status, version)
	return f
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

func TestProjectLotStageExecute(t *testing.T) {
	ctx := context.Background()

	t.Run("applies the stage and registers the message in one transaction", func(t *testing.T) {
		cmd := validCommand()
		f := fixtureWithView(t, cmd, lotview.StageFarm, lotview.StatusActive, 2)

		err := f.uc.Execute(ctx, cmd)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if f.tx.calls != 1 || f.views.usedOutside || f.processed.usedOutside {
			t.Errorf("expected one transaction around every access, got %d calls", f.tx.calls)
		}
		if len(f.processed.registered) != 1 || f.processed.registered[0] != cmd.MessageID || f.processed.types[0] != "lot.stage_registered" {
			t.Errorf("unexpected registered messages: %v %v", f.processed.registered, f.processed.types)
		}
		if len(f.views.found) != 1 || f.views.found[0] != cmd.LotID {
			t.Errorf("expected the lot view to be read once, got %v", f.views.found)
		}
		if len(f.views.saved) != 1 {
			t.Fatalf("expected one saved view, got %d", len(f.views.saved))
		}
		saved := f.views.saved[0]
		if saved.LotID() != cmd.LotID || saved.CurrentStage() != lotview.StageRoasting || saved.Version() != 3 {
			t.Errorf("unexpected saved view: stage %q, version %d", saved.CurrentStage(), saved.Version())
		}
		if !saved.UpdatedAt().Equal(cmd.OccurredAt) {
			t.Errorf("UpdatedAt: got %v, want %v", saved.UpdatedAt(), cmd.OccurredAt)
		}
	})

	t.Run("keeps the rest of the lot view", func(t *testing.T) {
		cmd := validCommand()
		f := fixtureWithView(t, cmd, lotview.StageFarm, lotview.StatusActive, 2)
		before := f.views.current

		if err := f.uc.Execute(ctx, cmd); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		saved := f.views.saved[0]
		if saved.Code() != before.Code() || saved.FarmID() != before.FarmID() ||
			saved.ProducerID() != before.ProducerID() || saved.CustodianID() != before.CustodianID() ||
			saved.Status() != before.Status() || !saved.CreatedAt().Equal(before.CreatedAt()) {
			t.Errorf("the saved view lost data: got %+v, want %+v", saved, before)
		}
		if before.CurrentStage() != lotview.StageFarm || before.Version() != 2 {
			t.Error("the view read from the repository must not be modified")
		}
	})

	t.Run("applies a version that skips numbers", func(t *testing.T) {
		cmd := validCommand()
		cmd.Version = 9
		f := fixtureWithView(t, cmd, lotview.StageFarm, lotview.StatusActive, 2)

		err := f.uc.Execute(ctx, cmd)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(f.views.saved) != 1 || f.views.saved[0].Version() != 9 {
			t.Errorf("expected version 9 to be saved, got %+v", f.views.saved)
		}
	})

	t.Run("invalidates the cache after the transaction", func(t *testing.T) {
		cmd := validCommand()
		f := fixtureWithView(t, cmd, lotview.StageFarm, lotview.StatusActive, 2)

		err := f.uc.Execute(ctx, cmd)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(f.cache.deleted) != 1 || f.cache.deleted[0] != cmd.LotID {
			t.Errorf("expected the lot to be evicted, got %v", f.cache.deleted)
		}
		if f.cache.deletedInsideTx {
			t.Error("the cache must be invalidated after the commit")
		}
	})

	t.Run("ignores a cache invalidation failure", func(t *testing.T) {
		cmd := validCommand()
		f := fixtureWithView(t, cmd, lotview.StageFarm, lotview.StatusActive, 2)
		f.cache.deleteErr = errBackend

		if err := f.uc.Execute(ctx, cmd); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("skips a message that was already processed", func(t *testing.T) {
		cmd := validCommand()
		f := fixtureWithView(t, cmd, lotview.StageFarm, lotview.StatusActive, 2)
		f.processed.first = false

		err := f.uc.Execute(ctx, cmd)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(f.views.found) != 0 || len(f.views.saved) != 0 {
			t.Errorf("a duplicate message must not touch the lot view, got %d reads and %d saves", len(f.views.found), len(f.views.saved))
		}
	})

	t.Run("ignores an event that is not newer than the stored version", func(t *testing.T) {
		for _, current := range []int{3, 5} {
			cmd := validCommand()
			f := fixtureWithView(t, cmd, lotview.StageRoasting, lotview.StatusActive, current)

			err := f.uc.Execute(ctx, cmd)

			if err != nil {
				t.Fatalf("current %d: unexpected error: %v", current, err)
			}
			if len(f.views.saved) != 0 {
				t.Errorf("current %d: an old event must not overwrite the view", current)
			}
		}
	})

	t.Run("ignores an event for a voided lot", func(t *testing.T) {
		cmd := validCommand()
		f := fixtureWithView(t, cmd, lotview.StageFarm, lotview.StatusVoided, 2)

		err := f.uc.Execute(ctx, cmd)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(f.views.saved) != 0 {
			t.Error("a voided lot must not change")
		}
	})
}

func TestProjectLotStageRejectsInvalidMessages(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(c *projectlotstage.Command)
	}{
		{"default message ID", func(c *projectlotstage.Command) { c.MessageID = uuid.Nil }},
		{"empty event type", func(c *projectlotstage.Command) { c.EventType = "  " }},
		{"default lot ID", func(c *projectlotstage.Command) { c.LotID = uuid.Nil }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture()
			cmd := validCommand()
			tt.mutate(&cmd)

			err := f.uc.Execute(context.Background(), cmd)

			assertCode(t, err, exception.CodeEventInvalidPayload)
			if f.tx.calls != 0 || len(f.cache.deleted) != 0 {
				t.Errorf("expected no transaction or eviction, got %d and %v", f.tx.calls, f.cache.deleted)
			}
		})
	}
}

func TestProjectLotStageRejectsAnInvalidStage(t *testing.T) {
	for _, stage := range []lotview.Stage{lotview.StageNone, lotview.StageUnknown, lotview.Stage("drying")} {
		t.Run(string(stage), func(t *testing.T) {
			cmd := validCommand()
			cmd.Stage = stage
			f := fixtureWithView(t, cmd, lotview.StageFarm, lotview.StatusActive, 2)

			err := f.uc.Execute(context.Background(), cmd)

			assertCode(t, err, exception.CodeLotViewInvalidStage)
			ex, _ := exception.From(err)
			if ex.Kind() != exception.KindValidation {
				t.Errorf("Kind: got %v, want %v so the message is discarded", ex.Kind(), exception.KindValidation)
			}
			if len(f.views.saved) != 0 || len(f.cache.deleted) != 0 {
				t.Errorf("expected no save or eviction, got %d and %v", len(f.views.saved), f.cache.deleted)
			}
		})
	}
}

func TestProjectLotStageWithoutLotViewIsRetryable(t *testing.T) {
	f := newFixture()
	cmd := validCommand()

	err := f.uc.Execute(context.Background(), cmd)

	assertCode(t, err, exception.CodeLotViewNotFound)
	ex, _ := exception.From(err)
	if ex.Kind() == exception.KindValidation {
		t.Error("a missing lot view must not be a validation error, or the message would be discarded instead of retried")
	}
	if len(f.views.saved) != 0 || len(f.cache.deleted) != 0 {
		t.Errorf("expected no save or eviction, got %d and %v", len(f.views.saved), f.cache.deleted)
	}
}

func TestProjectLotStageWrapsStorageFailures(t *testing.T) {
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
			cmd := validCommand()
			f := fixtureWithView(t, cmd, lotview.StageFarm, lotview.StatusActive, 2)
			tt.setup(f)

			err := f.uc.Execute(context.Background(), cmd)

			assertCode(t, err, exception.CodeInternal)
			if !errors.Is(err, errBackend) {
				t.Errorf("the cause must be preserved, got %v", err)
			}
			if len(f.cache.deleted) != 0 {
				t.Errorf("a failed projection must not evict the cache, got %v", f.cache.deleted)
			}
		})
	}
}
