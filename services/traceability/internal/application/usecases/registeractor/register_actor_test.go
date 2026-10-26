package registeractor_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"rutadeorigen/traceability/internal/application/usecases/registeractor"
	"rutadeorigen/traceability/internal/domain/actor"
	"rutadeorigen/traceability/internal/platform/exception"
)

var errInfra = errors.New("infrastructure failure")

type fakeActors struct {
	byUserID   map[string]*actor.Actor
	findErr    error
	findErrOnN int
	saveErr    error
	raceWinner *actor.Actor
	findCalls  int
	saveCalls  int
	saved      []*actor.Actor
}

func (r *fakeActors) FindByID(_ context.Context, _ uuid.UUID) (*actor.Actor, error) {
	return actor.Default(), nil
}

func (r *fakeActors) FindByUserID(_ context.Context, userID string) (*actor.Actor, error) {
	r.findCalls++
	if r.findErr != nil && (r.findErrOnN == 0 || r.findErrOnN == r.findCalls) {
		return actor.Default(), r.findErr
	}
	if a, ok := r.byUserID[userID]; ok {
		return a, nil
	}
	return actor.Default(), nil
}

func (r *fakeActors) Save(_ context.Context, a *actor.Actor) error {
	r.saveCalls++
	if r.raceWinner != nil {
		r.byUserID[r.raceWinner.UserID()] = r.raceWinner
		return exception.New(exception.CodeActorAlreadyExists)
	}
	if r.saveErr != nil {
		return r.saveErr
	}
	r.byUserID[a.UserID()] = a
	r.saved = append(r.saved, a)
	return nil
}

func newActors() *fakeActors {
	return &fakeActors{byUserID: map[string]*actor.Actor{}}
}

func mustActor(t *testing.T, userID string) *actor.Actor {
	t.Helper()
	a, err := actor.New(userID, actor.TypeProducer, "Finca La Esperanza SAS", userID+"@example.com")
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func validCommand() registeractor.Command {
	return registeractor.Command{
		UserID: "user_01",
		Email:  "Producer@Example.com",
		Type:   actor.TypeProducer,
		Name:   "Finca La Esperanza SAS",
	}
}

func assertCode(t *testing.T, err error, code exception.Code) {
	t.Helper()
	if !exception.HasCode(err, code) {
		t.Errorf("got error %v, want code %s", err, code)
	}
}

func assertInternal(t *testing.T, err error) {
	t.Helper()
	e, ok := exception.From(err)
	if !ok || e.Kind() != exception.KindInternal {
		t.Errorf("expected an internal error, got %v", err)
	}
	if !errors.Is(err, errInfra) {
		t.Error("expected the original cause to be kept")
	}
}

func assertEmptyResult(t *testing.T, got registeractor.Result) {
	t.Helper()
	if got.Created {
		t.Error("expected Created to be false")
	}
	if got.Actor != nil {
		t.Error("expected no actor in the result")
	}
}

func TestRegisterActorCreates(t *testing.T) {
	actors := newActors()
	uc := registeractor.New(actors)

	got, err := uc.Execute(context.Background(), validCommand())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Run("marks the result as created", func(t *testing.T) {
		if !got.Created {
			t.Error("expected Created to be true")
		}
	})

	t.Run("returns the saved actor", func(t *testing.T) {
		if got.Actor.IsDefault() {
			t.Fatal("expected a real actor")
		}
		if len(actors.saved) != 1 || actors.saved[0] != got.Actor {
			t.Error("expected the returned actor to be the persisted one")
		}
	})

	t.Run("links the actor to the session user", func(t *testing.T) {
		if got.Actor.UserID() != "user_01" {
			t.Errorf("got user id %q, want %q", got.Actor.UserID(), "user_01")
		}
	})

	t.Run("normalizes the email", func(t *testing.T) {
		if got.Actor.Email() != "producer@example.com" {
			t.Errorf("got email %q", got.Actor.Email())
		}
	})

	t.Run("creates an active actor with the requested data", func(t *testing.T) {
		if !got.Actor.IsActive() {
			t.Error("expected an active actor")
		}
		if got.Actor.Type() != actor.TypeProducer {
			t.Errorf("got type %q", got.Actor.Type())
		}
		if got.Actor.Name() != "Finca La Esperanza SAS" {
			t.Errorf("got name %q", got.Actor.Name())
		}
	})
}

func TestRegisterActorTrimsUserID(t *testing.T) {
	actors := newActors()
	cmd := validCommand()
	cmd.UserID = "  user_01  "

	got, err := registeractor.New(actors).Execute(context.Background(), cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Actor.UserID() != "user_01" {
		t.Errorf("got user id %q, want %q", got.Actor.UserID(), "user_01")
	}
}

func TestRegisterActorIsIdempotent(t *testing.T) {
	actors := newActors()
	existing := mustActor(t, "user_01")
	actors.byUserID["user_01"] = existing

	cmd := validCommand()
	cmd.Name = "Another Name"
	cmd.Type = actor.TypeBrand

	got, err := registeractor.New(actors).Execute(context.Background(), cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Run("returns the existing actor", func(t *testing.T) {
		if got.Actor != existing {
			t.Error("expected the existing actor")
		}
	})

	t.Run("is not reported as created", func(t *testing.T) {
		if got.Created {
			t.Error("expected Created to be false")
		}
	})

	t.Run("does not save", func(t *testing.T) {
		if actors.saveCalls != 0 {
			t.Errorf("got %d save calls, want 0", actors.saveCalls)
		}
	})

	t.Run("keeps the stored data", func(t *testing.T) {
		if got.Actor.Name() != "Finca La Esperanza SAS" || got.Actor.Type() != actor.TypeProducer {
			t.Error("expected the stored actor to be unchanged")
		}
	})
}

func TestRegisterActorSecondCallReturnsSameActor(t *testing.T) {
	actors := newActors()
	uc := registeractor.New(actors)

	first, err := uc.Execute(context.Background(), validCommand())
	if err != nil {
		t.Fatal(err)
	}
	second, err := uc.Execute(context.Background(), validCommand())
	if err != nil {
		t.Fatal(err)
	}

	if first.Actor.ID() != second.Actor.ID() {
		t.Error("expected the same actor on both calls")
	}
	if !first.Created || second.Created {
		t.Errorf("got created %v then %v, want true then false", first.Created, second.Created)
	}
	if actors.saveCalls != 1 {
		t.Errorf("got %d save calls, want 1", actors.saveCalls)
	}
}

func TestRegisterActorRequiresUserID(t *testing.T) {
	for name, userID := range map[string]string{"empty": "", "spaces": "   "} {
		t.Run(name, func(t *testing.T) {
			actors := newActors()
			cmd := validCommand()
			cmd.UserID = userID

			got, err := registeractor.New(actors).Execute(context.Background(), cmd)

			assertCode(t, err, exception.CodeUnauthorized)
			assertEmptyResult(t, got)
			if actors.findCalls != 0 || actors.saveCalls != 0 {
				t.Error("expected no repository access")
			}
		})
	}
}

func TestRegisterActorValidation(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*registeractor.Command)
		code   exception.Code
	}{
		{"unknown type", func(c *registeractor.Command) { c.Type = actor.TypeUnknown }, exception.CodeActorInvalidType},
		{"invalid type", func(c *registeractor.Command) { c.Type = actor.Type("farmer") }, exception.CodeActorInvalidType},
		{"short name", func(c *registeractor.Command) { c.Name = "ab" }, exception.CodeActorInvalidName},
		{"empty name", func(c *registeractor.Command) { c.Name = "   " }, exception.CodeActorInvalidName},
		{"empty email", func(c *registeractor.Command) { c.Email = "" }, exception.CodeActorInvalidEmail},
		{"blank email", func(c *registeractor.Command) { c.Email = "   " }, exception.CodeActorInvalidEmail},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actors := newActors()
			cmd := validCommand()
			tt.mutate(&cmd)

			got, err := registeractor.New(actors).Execute(context.Background(), cmd)

			assertCode(t, err, tt.code)
			assertEmptyResult(t, got)
			if actors.saveCalls != 0 {
				t.Error("expected nothing to be saved")
			}
		})
	}
}

func TestRegisterActorConflictReturnsWinner(t *testing.T) {
	actors := newActors()
	winner := mustActor(t, "user_01")
	actors.raceWinner = winner

	got, err := registeractor.New(actors).Execute(context.Background(), validCommand())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Run("returns the actor created concurrently", func(t *testing.T) {
		if got.Actor != winner {
			t.Error("expected the concurrent actor")
		}
	})

	t.Run("is not reported as created", func(t *testing.T) {
		if got.Created {
			t.Error("expected Created to be false")
		}
	})

	t.Run("looks the actor up again", func(t *testing.T) {
		if actors.findCalls != 2 {
			t.Errorf("got %d find calls, want 2", actors.findCalls)
		}
	})
}

func TestRegisterActorConflictWithoutActor(t *testing.T) {
	actors := newActors()
	actors.saveErr = exception.New(exception.CodeActorAlreadyExists)

	got, err := registeractor.New(actors).Execute(context.Background(), validCommand())

	e, ok := exception.From(err)
	if !ok || e.Kind() != exception.KindInternal {
		t.Errorf("expected an internal error, got %v", err)
	}
	assertEmptyResult(t, got)
}

func TestRegisterActorConflictLookupFails(t *testing.T) {
	actors := newActors()
	actors.saveErr = exception.New(exception.CodeActorAlreadyExists)
	actors.findErr = errInfra
	actors.findErrOnN = 2

	got, err := registeractor.New(actors).Execute(context.Background(), validCommand())

	assertInternal(t, err)
	assertEmptyResult(t, got)
}

func TestRegisterActorFindFails(t *testing.T) {
	actors := newActors()
	actors.findErr = errInfra

	got, err := registeractor.New(actors).Execute(context.Background(), validCommand())

	assertInternal(t, err)
	assertEmptyResult(t, got)
	if actors.saveCalls != 0 {
		t.Error("expected nothing to be saved")
	}
}

func TestRegisterActorSaveFails(t *testing.T) {
	actors := newActors()
	actors.saveErr = errInfra

	got, err := registeractor.New(actors).Execute(context.Background(), validCommand())

	assertInternal(t, err)
	assertEmptyResult(t, got)
}

func TestRegisterActorKeepsDomainErrorsFromRepository(t *testing.T) {
	actors := newActors()
	actors.saveErr = exception.New(exception.CodeForbidden)

	_, err := registeractor.New(actors).Execute(context.Background(), validCommand())

	assertCode(t, err, exception.CodeForbidden)
}
