package requestcontext_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"rutadeorigen/lib/requestcontext"
)

type otherKey int

func TestActorID(t *testing.T) {
	t.Run("returns the stored actor", func(t *testing.T) {
		id := uuid.New()
		ctx := requestcontext.WithActorID(context.Background(), id)

		if got := requestcontext.ActorID(ctx); got != id {
			t.Errorf("got %s, want %s", got, id)
		}
	})

	t.Run("empty context returns uuid.Nil", func(t *testing.T) {
		if got := requestcontext.ActorID(context.Background()); got != uuid.Nil {
			t.Errorf("got %s, want %s", got, uuid.Nil)
		}
	})

	t.Run("last stored actor wins", func(t *testing.T) {
		first, second := uuid.New(), uuid.New()
		ctx := requestcontext.WithActorID(context.Background(), first)
		ctx = requestcontext.WithActorID(ctx, second)

		if got := requestcontext.ActorID(ctx); got != second {
			t.Errorf("got %s, want %s", got, second)
		}
	})
}

func TestCorrelationID(t *testing.T) {
	t.Run("returns the stored correlation ID", func(t *testing.T) {
		ctx := requestcontext.WithCorrelationID(context.Background(), "req-123")

		if got := requestcontext.CorrelationID(ctx); got != "req-123" {
			t.Errorf("got %q, want %q", got, "req-123")
		}
	})

	t.Run("empty context returns empty string", func(t *testing.T) {
		if got := requestcontext.CorrelationID(context.Background()); got != "" {
			t.Errorf("got %q, want empty string", got)
		}
	})
}

func TestValuesDoNotOverlap(t *testing.T) {
	actor := uuid.New()
	ctx := requestcontext.WithActorID(context.Background(), actor)
	ctx = requestcontext.WithCorrelationID(ctx, "corr-1")
	ctx = requestcontext.WithUserID(ctx, "user_01H")

	if got := requestcontext.ActorID(ctx); got != actor {
		t.Fatalf("ActorID = %v, want %v", got, actor)
	}
	if got := requestcontext.CorrelationID(ctx); got != "corr-1" {
		t.Fatalf("CorrelationID = %q, want %q", got, "corr-1")
	}
	if got := requestcontext.UserID(ctx); got != "user_01H" {
		t.Fatalf("UserID = %q, want %q", got, "user_01H")
	}

	onlyCorrelation := requestcontext.WithCorrelationID(context.Background(), "req-456")
	if got := requestcontext.ActorID(onlyCorrelation); got != uuid.Nil {
		t.Errorf("storing a correlation ID must not set an actor, got %s", got)
	}
}

func TestOtherPackagesCannotOverrideKeys(t *testing.T) {
	// Another package using the same underlying values with its own key type
	// must not be read as an actor or a correlation ID.
	ctx := context.WithValue(context.Background(), otherKey(0), uuid.New())
	ctx = context.WithValue(ctx, otherKey(1), "foreign")

	if got := requestcontext.ActorID(ctx); got != uuid.Nil {
		t.Errorf("ActorID: got %s, want %s", got, uuid.Nil)
	}
	if got := requestcontext.CorrelationID(ctx); got != "" {
		t.Errorf("CorrelationID: got %q, want empty string", got)
	}
}

func TestUserIDRoundTrip(t *testing.T) {
	ctx := requestcontext.WithUserID(context.Background(), "user_01H")

	if got := requestcontext.UserID(ctx); got != "user_01H" {
		t.Fatalf("UserID = %q, want %q", got, "user_01H")
	}
}

func TestUserIDMissingReturnsEmpty(t *testing.T) {
	if got := requestcontext.UserID(context.Background()); got != "" {
		t.Fatalf("UserID = %q, want empty", got)
	}
}

func TestEmailRoundTrip(t *testing.T) {
	ctx := requestcontext.WithEmail(context.Background(), "producer@example.com")

	if got := requestcontext.Email(ctx); got != "producer@example.com" {
		t.Fatalf("Email = %q, want %q", got, "producer@example.com")
	}
}

func TestEmailMissingReturnsEmpty(t *testing.T) {
	if got := requestcontext.Email(context.Background()); got != "" {
		t.Fatalf("Email = %q, want empty", got)
	}
}

func TestEmailAndUserIDAreIndependent(t *testing.T) {
	ctx := requestcontext.WithUserID(context.Background(), "user_01H")

	if got := requestcontext.Email(ctx); got != "" {
		t.Errorf("Email = %q, want empty", got)
	}

	ctx = requestcontext.WithEmail(ctx, "producer@example.com")
	if got := requestcontext.UserID(ctx); got != "user_01H" {
		t.Errorf("UserID = %q, want %q", got, "user_01H")
	}
}

func TestFirstNameRoundTrip(t *testing.T) {
	ctx := requestcontext.WithFirstName(context.Background(), "Juan")

	if got := requestcontext.FirstName(ctx); got != "Juan" {
		t.Fatalf("FirstName = %q, want %q", got, "Juan")
	}
}

func TestFirstNameMissingReturnsEmpty(t *testing.T) {
	if got := requestcontext.FirstName(context.Background()); got != "" {
		t.Fatalf("FirstName = %q, want empty", got)
	}
}

func TestLastNameRoundTrip(t *testing.T) {
	ctx := requestcontext.WithLastName(context.Background(), "Ramírez")

	if got := requestcontext.LastName(ctx); got != "Ramírez" {
		t.Fatalf("LastName = %q, want %q", got, "Ramírez")
	}
}

func TestLastNameMissingReturnsEmpty(t *testing.T) {
	if got := requestcontext.LastName(context.Background()); got != "" {
		t.Fatalf("LastName = %q, want empty", got)
	}
}

func TestNamesAreIndependentFromEachOtherAndFromOtherValues(t *testing.T) {
	ctx := requestcontext.WithFirstName(context.Background(), "Juan")

	if got := requestcontext.LastName(ctx); got != "" {
		t.Errorf("LastName = %q, want empty", got)
	}
	if got := requestcontext.Email(ctx); got != "" {
		t.Errorf("Email = %q, want empty", got)
	}

	ctx = requestcontext.WithLastName(ctx, "Ramírez")
	ctx = requestcontext.WithEmail(ctx, "producer@example.com")
	ctx = requestcontext.WithUserID(ctx, "user_01H")

	if got := requestcontext.FirstName(ctx); got != "Juan" {
		t.Errorf("FirstName = %q, want %q", got, "Juan")
	}
	if got := requestcontext.LastName(ctx); got != "Ramírez" {
		t.Errorf("LastName = %q, want %q", got, "Ramírez")
	}
	if got := requestcontext.Email(ctx); got != "producer@example.com" {
		t.Errorf("Email = %q, want %q", got, "producer@example.com")
	}
	if got := requestcontext.UserID(ctx); got != "user_01H" {
		t.Errorf("UserID = %q, want %q", got, "user_01H")
	}
}

func TestLastStoredNameWins(t *testing.T) {
	ctx := requestcontext.WithFirstName(context.Background(), "Juan")
	ctx = requestcontext.WithFirstName(ctx, "Ana")

	if got := requestcontext.FirstName(ctx); got != "Ana" {
		t.Errorf("FirstName = %q, want %q", got, "Ana")
	}
}
