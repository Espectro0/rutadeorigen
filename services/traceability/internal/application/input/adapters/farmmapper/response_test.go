package farmmapper_test

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"rutadeorigen/traceability/internal/application/input/adapters/farmmapper"
	"rutadeorigen/traceability/internal/application/input/dto/response"
	"rutadeorigen/traceability/internal/domain/farm"
)

func restoredFarm(name string) *farm.Farm {
	return farm.Restore(uuid.New(), uuid.New(), name, 1650, time.Date(2026, time.September, 20, 14, 30, 0, 0, time.UTC))
}

func TestToResponse(t *testing.T) {
	t.Run("maps every field", func(t *testing.T) {
		f := restoredFarm("La Esperanza")

		got := farmmapper.ToResponse(f)

		want := response.Farm{
			ID:         f.ID().String(),
			ProducerID: f.ProducerID().String(),
			Name:       "La Esperanza",
			Altitude:   1650,
			CreatedAt:  "2026-09-20T14:30:00Z",
		}
		if got != want {
			t.Errorf("got %+v, want %+v", got, want)
		}
	})

	t.Run("default farm returns an empty response", func(t *testing.T) {
		if got := farmmapper.ToResponse(farm.Default()); got != (response.Farm{}) {
			t.Errorf("got %+v, want empty response", got)
		}
	})

	t.Run("nil farm returns an empty response", func(t *testing.T) {
		if got := farmmapper.ToResponse(nil); got != (response.Farm{}) {
			t.Errorf("got %+v, want empty response", got)
		}
	})
}

func TestToList(t *testing.T) {
	t.Run("maps every farm in order", func(t *testing.T) {
		first, second := restoredFarm("La Esperanza"), restoredFarm("El Mirador")

		got := farmmapper.ToList([]*farm.Farm{first, second})

		if len(got.Items) != 2 {
			t.Fatalf("got %d items, want 2", len(got.Items))
		}
		if got.Items[0].ID != first.ID().String() || got.Items[1].ID != second.ID().String() {
			t.Error("unexpected order or content")
		}
	})

	t.Run("skips default and nil farms", func(t *testing.T) {
		f := restoredFarm("La Esperanza")

		got := farmmapper.ToList([]*farm.Farm{farm.Default(), f, nil})

		if len(got.Items) != 1 || got.Items[0].ID != f.ID().String() {
			t.Errorf("got %+v, want only the real farm", got.Items)
		}
	})

	t.Run("an empty or nil list returns empty, non-nil items", func(t *testing.T) {
		for _, farms := range [][]*farm.Farm{{}, nil} {
			got := farmmapper.ToList(farms)
			if got.Items == nil || len(got.Items) != 0 {
				t.Errorf("got %+v, want empty, non-nil items", got.Items)
			}
		}
	})
}
