package memory_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"rutadeorigen/traceability/internal/domain/farm"
	"rutadeorigen/traceability/internal/infrastructure/output/repositories/structured/memory"
	"rutadeorigen/traceability/internal/platform/exception"
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

func TestFarmRepositorySave(t *testing.T) {
	ctx := context.Background()

	t.Run("a saved farm can be found", func(t *testing.T) {
		repo := memory.NewFarmRepository()
		f := newFarm()

		if err := repo.Save(ctx, f); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got, err := repo.FindByID(ctx, f.ID())
		if err != nil || got != f {
			t.Errorf("got %v, %v; want the saved farm", got, err)
		}
	})

	t.Run("rejects a default or nil farm", func(t *testing.T) {
		repo := memory.NewFarmRepository()

		for _, f := range []*farm.Farm{farm.Default(), nil} {
			err := repo.Save(ctx, f)

			e, ok := exception.From(err)
			if !ok || e.Kind() != exception.KindInternal {
				t.Errorf("got %v, want an internal error", err)
			}
		}

		got, _ := repo.List(ctx)
		if len(got) != 0 {
			t.Error("nothing must be saved")
		}
	})
}

func TestFarmRepositoryList(t *testing.T) {
	ctx := context.Background()
	base := time.Date(2026, time.October, 1, 12, 0, 0, 0, time.UTC)

	older := farm.Restore(uuid.New(), uuid.New(), "La Esperanza", 1650, base)
	newer := farm.Restore(uuid.New(), uuid.New(), "El Mirador", 1800, base.Add(time.Hour))
	newest := farm.Restore(uuid.New(), uuid.New(), "La Otra Finca", 1500, base.Add(2*time.Hour))

	t.Run("returns the farms of every producer, newest first", func(t *testing.T) {
		repo := memory.NewFarmRepository(older, newest, newer)

		got, err := repo.List(ctx)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 3 || got[0] != newest || got[1] != newer || got[2] != older {
			t.Errorf("got %v, want [newest, newer, older]", got)
		}
	})

	t.Run("ties on the date are ordered by id, descending", func(t *testing.T) {
		a := farm.Restore(uuid.MustParse("00000000-0000-0000-0000-000000000001"), uuid.New(), "Finca A", 1650, base)
		b := farm.Restore(uuid.MustParse("00000000-0000-0000-0000-000000000002"), uuid.New(), "Finca B", 1650, base)
		repo := memory.NewFarmRepository(a, b)

		got, _ := repo.List(ctx)

		if len(got) != 2 || got[0] != b || got[1] != a {
			t.Errorf("got %v, want [b, a]", got)
		}
	})

	t.Run("without farms it returns an empty, non-nil list", func(t *testing.T) {
		repo := memory.NewFarmRepository()

		got, err := repo.List(ctx)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got == nil || len(got) != 0 {
			t.Errorf("got %v, want an empty, non-nil list", got)
		}
	})
}
