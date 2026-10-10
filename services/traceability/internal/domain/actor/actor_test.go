package actor_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"rutadeorigen/traceability/internal/domain/actor"
)

func TestNewValid(t *testing.T) {
	a, err := actor.New(" user_01 ", actor.TypeProducer, "  Finca La Esperanza SAS  ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.UserID() != "user_01" || a.Name() != "Finca La Esperanza SAS" {
		t.Errorf("expected trimmed values, got %q and %q", a.UserID(), a.Name())
	}
	if a.Type() != actor.TypeProducer || !a.IsProducer() {
		t.Error("expected a producer")
	}
	if !a.IsActive() || a.Status() != actor.StatusActive {
		t.Error("a new actor must be active")
	}
	if a.ID() == uuid.Nil || a.CreatedAt().IsZero() {
		t.Error("expected generated ID and creation date")
	}
}

func TestNewInvalid(t *testing.T) {
	tests := []struct {
		name      string
		userID    string
		actorType actor.Type
		actorName string
		want      error
	}{
		{"empty user ID", "   ", actor.TypeProducer, "Tostadora Andina", actor.ErrInvalidUserID},
		{"unknown actor type", "user_01", actor.TypeUnknown, "Tostadora Andina", actor.ErrInvalidType},
		{"invalid actor type", "user_01", actor.Type("admin"), "Tostadora Andina", actor.ErrInvalidType},
		{"name too short", "user_01", actor.TypeRoaster, "AB", actor.ErrInvalidName},
		{"name only spaces", "user_01", actor.TypeRoaster, "     ", actor.ErrInvalidName},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a, err := actor.New(tt.userID, tt.actorType, tt.actorName)
			if !errors.Is(err, tt.want) {
				t.Errorf("got %v, want %v", err, tt.want)
			}
			if !a.IsDefault() {
				t.Error("expected default actor on error")
			}
		})
	}
}

func TestRestore(t *testing.T) {
	id := uuid.New()
	createdAt := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

	a := actor.Restore(id, "user_02", actor.TypeRoaster, "Tostadora Andina",
		actor.StatusInactive, createdAt)

	if a.ID() != id || !a.CreatedAt().Equal(createdAt) {
		t.Error("expected the stored ID and creation date to be kept")
	}
	if a.IsActive() || a.IsProducer() {
		t.Error("expected an inactive roaster")
	}
}

func TestDefault(t *testing.T) {
	d := actor.Default()

	if !d.IsDefault() {
		t.Error("expected Default() to be the default actor")
	}
	if d.ID() != uuid.Nil || d.UserID() != "" || d.Name() != "" {
		t.Error("expected default values")
	}
	if d.Type() != actor.TypeUnknown || d.Status() != actor.StatusUnknown {
		t.Error("expected unknown type and status")
	}
	if d.IsActive() || d.IsProducer() {
		t.Error("the default actor must not be active nor a producer")
	}
	if d == actor.Default() {
		t.Error("expected each call to return a new copy")
	}

	created, _ := actor.New("user_01", actor.TypeProducer, "Tostadora Andina")
	if created.IsDefault() {
		t.Error("a new actor must not be the default actor")
	}

	var nilActor *actor.Actor
	if !nilActor.IsDefault() {
		t.Error("a nil actor must be treated as default")
	}
}
