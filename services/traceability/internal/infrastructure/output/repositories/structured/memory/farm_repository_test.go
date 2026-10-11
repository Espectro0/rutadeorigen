package memory_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"rutadeorigen/traceability/internal/domain/farm"
	"rutadeorigen/traceability/internal/infrastructure/output/repositories/structured/memory"
)

func newFarm() *farm.Farm {
	return farm.Restore(uuid.New(), uuid.New(), "Finca", 1800, time.Now().UTC())
}

func TestFarmRepositoryFindById(t *testing.T) {
	ctx := context.Background()

	t.Run("returns a seeded farm", func(t *testing.T) {
		f := newFarm()
		repo := memory.NewFarmRepository(f)

		got, err := repo.FindByID(ctx, f.ID())

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != f {
			t.Errorf("got %v, want the seeded farm", got)
		}
	})

	t.Run("unknown id returns the default farm", func(t *testing.T) {
		repo := memory.NewFarmRepository(newFarm())

		got, err := repo.FindByID(ctx, uuid.New())

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got == nil || !got.IsDefault() {
			t.Errorf("got %v, want farm.Default()", got)
		}
	})

	t.Run("seed ignores nil and default farms", func(t *testing.T) {
		repo := memory.NewFarmRepository(nil, farm.Default())

		got, err := repo.FindByID(ctx, uuid.Nil)

		if err != nil || got == nil || !got.IsDefault() {
			t.Errorf("got %v, %v; want farm.Default(), nil", got, err)
		}
	})
}
