package memory_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"rutadeorigen/traceability/internal/domain/actor"
	"rutadeorigen/traceability/internal/infrastructure/output/repositories/structured/memory"
	"rutadeorigen/traceability/internal/platform/exception"
)

func newActor() *actor.Actor {
	return actor.Restore(uuid.New(), "user-"+uuid.NewString(), actor.TypeProducer,
		"Productor", "productor@example.com", actor.StatusActive, time.Now().UTC())
}

func TestActorRepositoryFindById(t *testing.T) {
	ctx := context.Background()

	t.Run("returns a seeded actor", func(t *testing.T) {
		a := newActor()
		repo := memory.NewActorRepository(a)

		got, err := repo.FindByID(ctx, a.ID())

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != a {
			t.Errorf("got %v, want the seeded actor", got)
		}
	})

	t.Run("unknown id returns the default actor", func(t *testing.T) {
		repo := memory.NewActorRepository(newActor())

		got, err := repo.FindByID(ctx, uuid.New())

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got == nil || !got.IsDefault() {
			t.Errorf("got %v, want actor.Default()", got)
		}
	})

	t.Run("empty repository returns the default actor", func(t *testing.T) {
		got, err := memory.NewActorRepository().FindByID(ctx, uuid.New())

		if err != nil || got == nil || !got.IsDefault() {
			t.Errorf("got %v, %v; want actor.Default(), nil", got, err)
		}
	})

	t.Run("seed ignores nil and default actors", func(t *testing.T) {
		repo := memory.NewActorRepository(nil, actor.Default())

		got, err := repo.FindByID(ctx, uuid.Nil)

		if err != nil || got == nil || !got.IsDefault() {
			t.Errorf("got %v, %v; want actor.Default(), nil", got, err)
		}
	})
}

func TestActorRepositoryFindByUserID(t *testing.T) {
	ctx := context.Background()

	t.Run("returns the actor with that user id", func(t *testing.T) {
		a := newActor()
		repo := memory.NewActorRepository(newActor(), a, newActor())

		got, err := repo.FindByUserID(ctx, a.UserID())

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != a {
			t.Errorf("got %v, want the actor with user id %q", got, a.UserID())
		}
	})

	t.Run("unknown user id returns the default actor", func(t *testing.T) {
		repo := memory.NewActorRepository(newActor())

		got, err := repo.FindByUserID(ctx, "user-unknown")

		if err != nil || got == nil || !got.IsDefault() {
			t.Errorf("got %v, %v; want actor.Default(), nil", got, err)
		}
	})

	t.Run("empty repository returns the default actor", func(t *testing.T) {
		got, err := memory.NewActorRepository().FindByUserID(ctx, "user-1")

		if err != nil || got == nil || !got.IsDefault() {
			t.Errorf("got %v, %v; want actor.Default(), nil", got, err)
		}
	})
}

func TestActorRepositorySave(t *testing.T) {
	ctx := context.Background()

	t.Run("stores the actor and finds it by id and by user id", func(t *testing.T) {
		repo := memory.NewActorRepository()
		a := newActor()

		if err := repo.Save(ctx, a); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		byID, err := repo.FindByID(ctx, a.ID())
		if err != nil || byID != a {
			t.Errorf("FindByID got %v, %v; want the saved actor, nil", byID, err)
		}
		byUserID, err := repo.FindByUserID(ctx, a.UserID())
		if err != nil || byUserID != a {
			t.Errorf("FindByUserID got %v, %v; want the saved actor, nil", byUserID, err)
		}
	})

	t.Run("saving the same actor again updates it", func(t *testing.T) {
		original := newActor()
		repo := memory.NewActorRepository(original)
		updated := actor.Restore(original.ID(), original.UserID(), original.Type(),
			original.Name(), "nuevo@example.com", original.Status(), original.CreatedAt())

		if err := repo.Save(ctx, updated); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got, err := repo.FindByID(ctx, original.ID())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Email() != "nuevo@example.com" {
			t.Errorf("email = %q, want %q", got.Email(), "nuevo@example.com")
		}
	})

	t.Run("rejects another actor with the same user id", func(t *testing.T) {
		existing := newActor()
		repo := memory.NewActorRepository(existing)
		duplicate := actor.Restore(uuid.New(), existing.UserID(), actor.TypeRoaster,
			"Tostadora", "otra@example.com", actor.StatusActive, time.Now().UTC())

		err := repo.Save(ctx, duplicate)

		if !exception.HasCode(err, exception.CodeActorAlreadyExists) {
			t.Fatalf("error = %v, want code %s", err, exception.CodeActorAlreadyExists)
		}
		got, findErr := repo.FindByID(ctx, duplicate.ID())
		if findErr != nil || !got.IsDefault() {
			t.Errorf("rejected actor must not be stored, got %v, %v", got, findErr)
		}
	})

	t.Run("rejects the default actor", func(t *testing.T) {
		err := memory.NewActorRepository().Save(ctx, actor.Default())

		e, ok := exception.From(err)
		if !ok || e.Kind() != exception.KindInternal {
			t.Errorf("error = %v, want an internal exception", err)
		}
	})
}
