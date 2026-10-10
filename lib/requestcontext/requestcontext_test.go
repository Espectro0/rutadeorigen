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
	id := uuid.New()
	ctx := requestcontext.WithActorID(context.Background(), id)
	ctx = requestcontext.WithCorrelationID(ctx, "req-123")

	if got := requestcontext.ActorID(ctx); got != id {
		t.Errorf("ActorID: got %s, want %s", got, id)
	}
	if got := requestcontext.CorrelationID(ctx); got != "req-123" {
		t.Errorf("CorrelationID: got %q, want %q", got, "req-123")
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
