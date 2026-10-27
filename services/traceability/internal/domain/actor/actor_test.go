package actor_test

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"rutadeorigen/traceability/internal/domain/actor"
	"rutadeorigen/traceability/internal/platform/exception"
)

func TestNewValid(t *testing.T) {
	a, err := actor.New(" user_01 ", actor.TypeProducer, "  Finca La Esperanza SAS  ", "  User@Example.com  ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.UserID() != "user_01" || a.Name() != "Finca La Esperanza SAS" {
		t.Errorf("expected trimmed values, got %q and %q", a.UserID(), a.Name())
	}
	if a.Email() != "user@example.com" {
		t.Errorf("expected a trimmed lowercase email, got %q", a.Email())
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
		email     string
		wantCode  exception.Code
	}{
		{"empty user ID", "   ", actor.TypeProducer, "Tostadora Andina", "user@example.com", exception.CodeActorInvalidUserID},
		{"unknown actor type", "user_01", actor.TypeUnknown, "Tostadora Andina", "user@example.com", exception.CodeActorInvalidType},
		{"invalid actor type", "user_01", actor.Type("admin"), "Tostadora Andina", "user@example.com", exception.CodeActorInvalidType},
		{"name too short", "user_01", actor.TypeRoaster, "AB", "user@example.com", exception.CodeActorInvalidName},
		{"name only spaces", "user_01", actor.TypeRoaster, "     ", "user@example.com", exception.CodeActorInvalidName},
		{"empty email", "user_01", actor.TypeRoaster, "Tostadora Andina", "", exception.CodeActorInvalidEmail},
		{"email only spaces", "user_01", actor.TypeRoaster, "Tostadora Andina", "     ", exception.CodeActorInvalidEmail},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a, err := actor.New(tt.userID, tt.actorType, tt.actorName, tt.email)
			if !exception.HasCode(err, tt.wantCode) {
				t.Errorf("got %v, want code %q", err, tt.wantCode)
			}
			if !a.IsDefault() {
				t.Error("expected default actor on error")
			}
		})
	}
}

func TestNewInvalidNameCarriesLengthLimits(t *testing.T) {
	_, err := actor.New("user_01", actor.TypeRoaster, "AB", "user@example.com")

	ex, ok := exception.From(err)
	if !ok {
		t.Fatalf("expected an exception, got %v", err)
	}
	if ex.Params()["min"] != 3 || ex.Params()["max"] != 100 {
		t.Errorf("params = %v, want min=3 max=100", ex.Params())
	}
}

func TestRestore(t *testing.T) {
	id := uuid.New()
	createdAt := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

	a := actor.Restore(id, "user_02", actor.TypeRoaster, "Tostadora Andina", "user_02@example.com",
		actor.StatusInactive, createdAt)

	if a.ID() != id || !a.CreatedAt().Equal(createdAt) {
		t.Error("expected the stored ID and creation date to be kept")
	}
	if a.Email() != "user_02@example.com" {
		t.Errorf("email = %q, want %q", a.Email(), "user_02@example.com")
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
	if d.ID() != uuid.Nil || d.UserID() != "" || d.Name() != "" || d.Email() != "" {
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

	created, _ := actor.New("user_01", actor.TypeProducer, "Tostadora Andina", "user@example.com")
	if created.IsDefault() {
		t.Error("a new actor must not be the default actor")
	}

	var nilActor *actor.Actor
	if !nilActor.IsDefault() {
		t.Error("a nil actor must be treated as default")
	}
}
