package projectactorregistered_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"rutadeorigen/qr/internal/application/usecases/projectactorregistered"
	"rutadeorigen/qr/internal/domain/actorview"
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

type fakeActors struct {
	tx          *fakeTx
	saveErr     error
	saved       []*actorview.ActorView
	usedOutside bool
}

func (a *fakeActors) FindByID(_ context.Context, _ uuid.UUID) (*actorview.ActorView, error) {
	return actorview.Default(), nil
}

func (a *fakeActors) FindByUserID(_ context.Context, _ string) (*actorview.ActorView, error) {
	return actorview.Default(), nil
}

func (a *fakeActors) Save(_ context.Context, view *actorview.ActorView) error {
	a.usedOutside = a.usedOutside || !a.tx.active
	if a.saveErr != nil {
		return a.saveErr
	}
	a.saved = append(a.saved, view)
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

type fixture struct {
	tx        *fakeTx
	actors    *fakeActors
	processed *fakeProcessed
	uc        *projectactorregistered.ProjectActorRegistered
}

func newFixture() *fixture {
	tx := &fakeTx{}
	f := &fixture{
		tx:        tx,
		actors:    &fakeActors{tx: tx},
		processed: &fakeProcessed{tx: tx, first: true},
	}
	f.uc = projectactorregistered.New(f.actors, f.processed, f.tx)
	return f
}

func validCommand() projectactorregistered.Command {
	return projectactorregistered.Command{
		MessageID:  uuid.New(),
		EventType:  "actor.registered",
		ActorID:    uuid.New(),
		UserID:     "user_01",
		Role:       actorview.RoleBrand,
		OccurredAt: time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC),
	}
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

func TestProjectActorRegisteredExecute(t *testing.T) {
	ctx := context.Background()

	t.Run("saves the actor view and registers the message in one transaction", func(t *testing.T) {
		cmd := validCommand()
		f := newFixture()

		if err := f.uc.Execute(ctx, cmd); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if f.tx.calls != 1 || f.actors.usedOutside || f.processed.usedOutside {
			t.Errorf("expected one transaction around every access, got %d calls", f.tx.calls)
		}
		if len(f.processed.registered) != 1 || f.processed.registered[0] != cmd.MessageID || f.processed.types[0] != "actor.registered" {
			t.Errorf("unexpected registered messages: %v %v", f.processed.registered, f.processed.types)
		}
		if len(f.actors.saved) != 1 {
			t.Fatalf("expected one saved view, got %d", len(f.actors.saved))
		}
		saved := f.actors.saved[0]
		if saved.ActorID() != cmd.ActorID || saved.UserID() != "user_01" || saved.Role() != actorview.RoleBrand ||
			!saved.CreatedAt().Equal(cmd.OccurredAt) {
			t.Errorf("unexpected saved view %+v", saved)
		}
	})

	t.Run("ignores a message that was already processed", func(t *testing.T) {
		f := newFixture()
		f.processed.first = false

		if err := f.uc.Execute(ctx, validCommand()); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(f.actors.saved) != 0 {
			t.Errorf("a duplicate must not save, got %d saves", len(f.actors.saved))
		}
	})

	t.Run("rejects an incomplete command without opening a transaction", func(t *testing.T) {
		for name, mutate := range map[string]func(*projectactorregistered.Command){
			"no message id": func(c *projectactorregistered.Command) { c.MessageID = uuid.Nil },
			"no event type": func(c *projectactorregistered.Command) { c.EventType = "  " },
		} {
			t.Run(name, func(t *testing.T) {
				cmd := validCommand()
				mutate(&cmd)
				f := newFixture()

				assertCode(t, f.uc.Execute(ctx, cmd), exception.CodeEventInvalidPayload)
				if f.tx.calls != 0 {
					t.Errorf("no transaction expected, got %d", f.tx.calls)
				}
			})
		}
	})

	t.Run("reports invalid actor data as a validation error", func(t *testing.T) {
		for name, tt := range map[string]struct {
			mutate func(*projectactorregistered.Command)
			code   exception.Code
		}{
			"no actor id":  {func(c *projectactorregistered.Command) { c.ActorID = uuid.Nil }, exception.CodeActorViewInvalidID},
			"no user":      {func(c *projectactorregistered.Command) { c.UserID = " " }, exception.CodeActorViewInvalidUser},
			"unknown role": {func(c *projectactorregistered.Command) { c.Role = actorview.RoleUnknown }, exception.CodeActorViewInvalidRole},
		} {
			t.Run(name, func(t *testing.T) {
				cmd := validCommand()
				tt.mutate(&cmd)
				f := newFixture()

				assertCode(t, f.uc.Execute(ctx, cmd), tt.code)
				if len(f.actors.saved) != 0 {
					t.Error("an invalid actor must not be saved")
				}
			})
		}
	})

	t.Run("wraps repository failures", func(t *testing.T) {
		t.Run("processed event", func(t *testing.T) {
			f := newFixture()
			f.processed.err = errBackend

			err := f.uc.Execute(ctx, validCommand())

			if !errors.Is(err, errBackend) {
				t.Errorf("expected the backend error, got %v", err)
			}
		})
		t.Run("save", func(t *testing.T) {
			f := newFixture()
			f.actors.saveErr = errBackend

			err := f.uc.Execute(ctx, validCommand())

			if !errors.Is(err, errBackend) {
				t.Errorf("expected the backend error, got %v", err)
			}
		})
	})
}
