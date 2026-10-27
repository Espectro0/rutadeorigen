package actormapper_test

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"rutadeorigen/traceability/internal/application/input/adapters/actormapper"
	"rutadeorigen/traceability/internal/application/input/dto/response"
	"rutadeorigen/traceability/internal/domain/actor"
)

func restoredActor(status actor.Status) *actor.Actor {
	return actor.Restore(uuid.New(), "user_01", actor.TypeProducer, "Finca La Esperanza SAS", "producer@example.com", status, time.Date(2026, time.September, 20, 14, 30, 0, 0, time.UTC))
}

func TestToResponse(t *testing.T) {
	t.Run("maps every field", func(t *testing.T) {
		a := restoredActor(actor.StatusActive)

		got := actormapper.ToResponse(a)

		want := response.Actor{
			ID:        a.ID().String(),
			Type:      "producer",
			Name:      "Finca La Esperanza SAS",
			Email:     "producer@example.com",
			Status:    "active",
			CreatedAt: "2026-09-20T14:30:00Z",
		}
		if got != want {
			t.Errorf("got %+v, want %+v", got, want)
		}
	})

	t.Run("keeps the inactive status", func(t *testing.T) {
		if got := actormapper.ToResponse(restoredActor(actor.StatusInactive)); got.Status != "inactive" {
			t.Errorf("status: got %q, want inactive", got.Status)
		}
	})

	t.Run("default actor returns an empty response", func(t *testing.T) {
		if got := actormapper.ToResponse(actor.Default()); got != (response.Actor{}) {
			t.Errorf("got %+v, want empty response", got)
		}
	})

	t.Run("nil actor returns an empty response", func(t *testing.T) {
		if got := actormapper.ToResponse(nil); got != (response.Actor{}) {
			t.Errorf("got %+v, want empty response", got)
		}
	})

	t.Run("does not expose the session user id", func(t *testing.T) {
		got := actormapper.ToResponse(restoredActor(actor.StatusActive))
		if got.ID == "user_01" {
			t.Error("the response id must be the actor id, not the user id")
		}
	})
}
