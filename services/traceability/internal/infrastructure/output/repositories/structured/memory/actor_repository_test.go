package memory_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"rutadeorigen/traceability/internal/domain/actor"
	"rutadeorigen/traceability/internal/infrastructure/output/repositories/structured/memory"
)

func newActor() *actor.Actor {
	return actor.Restore(uuid.New(), "user-"+uuid.NewString(), actor.TypeProducer,
		"Productor", actor.StatusActive, time.Now().UTC())
}

func TestActorRepositoryFindById(t *testing.T) {
	ctx := context.Background()

	t.Run("returns a seeded actor", func(t *testing.T) {
		a := newActor()
		repo := memory.NewActorRepository(a)

		got, err := repo.FindById(ctx, a.ID())

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != a {
			t.Errorf("got %v, want the seeded actor", got)
		}
	})

	t.Run("unknown id returns the default actor", func(t *testing.T) {
		repo := memory.NewActorRepository(newActor())

		got, err := repo.FindById(ctx, uuid.New())

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got == nil || !got.IsDefault() {
			t.Errorf("got %v, want actor.Default()", got)
		}
	})

	t.Run("empty repository returns the default actor", func(t *testing.T) {
		got, err := memory.NewActorRepository().FindById(ctx, uuid.New())

		if err != nil || got == nil || !got.IsDefault() {
			t.Errorf("got %v, %v; want actor.Default(), nil", got, err)
		}
	})

	t.Run("seed ignores nil and default actors", func(t *testing.T) {
		repo := memory.NewActorRepository(nil, actor.Default())

		got, err := repo.FindById(ctx, uuid.Nil)

		if err != nil || got == nil || !got.IsDefault() {
			t.Errorf("got %v, %v; want actor.Default(), nil", got, err)
		}
	})
}
