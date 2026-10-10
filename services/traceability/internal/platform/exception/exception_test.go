package exception_test

import (
	"errors"
	"fmt"
	"testing"

	"rutadeorigen/traceability/internal/platform/exception"
)

func TestErrorWithoutCause(t *testing.T) {
	err := exception.NewValidation(exception.LayerDomain, "Invalid lot name", "lot name is empty")

	if got := err.Error(); got != "lot name is empty" {
		t.Errorf("got %q, want %q", got, "lot name is empty")
	}
}

func TestErrorWithCause(t *testing.T) {
	cause := errors.New("connection refused")
	err := exception.NewInternal(exception.LayerInfrastructure, "", "save lot", cause)

	if got := err.Error(); got != "save lot: connection refused" {
		t.Errorf("got %q, want %q", got, "save lot: connection refused")
	}
}

func TestUnwrap(t *testing.T) {
	cause := errors.New("connection refused")

	withCause := exception.NewInternal(exception.LayerInfrastructure, "", "save lot", cause)
	if !errors.Is(withCause, cause) {
		t.Error("expected errors.Is to find the original cause")
	}

	withoutCause := exception.NewNotFound(exception.LayerApplication, "Lot not found", "")
	if withoutCause.Unwrap() != nil {
		t.Error("expected Unwrap to return nil when there is no cause")
	}
}

func TestGetters(t *testing.T) {
	err := exception.NewConflict(exception.LayerApplication, "Lot already exists", "duplicate lot code")

	if err.UserMessage() != "Lot already exists" {
		t.Errorf("UserMessage: got %q", err.UserMessage())
	}
	if err.TechnicalMessage() != "duplicate lot code" {
		t.Errorf("TechnicalMessage: got %q", err.TechnicalMessage())
	}
	if err.Layer() != exception.LayerApplication {
		t.Errorf("Layer: got %v", err.Layer())
	}
	if err.Kind() != exception.KindConflict {
		t.Errorf("Kind: got %v", err.Kind())
	}
}

func TestFrom(t *testing.T) {
	t.Run("finds exception directly", func(t *testing.T) {
		original := exception.NewValidation(exception.LayerDomain, "Invalid stage", "")

		got, ok := exception.From(original)
		if !ok || got != original {
			t.Error("expected From to return the same exception")
		}
	})

	t.Run("finds exception wrapped with fmt.Errorf", func(t *testing.T) {
		original := exception.NewValidation(exception.LayerDomain, "Invalid stage", "")
		wrapped := fmt.Errorf("register stage: %w", original)

		got, ok := exception.From(wrapped)
		if !ok || got != original {
			t.Error("expected From to find the wrapped exception")
		}
	})

	t.Run("returns false for a plain error", func(t *testing.T) {
		if _, ok := exception.From(errors.New("plain error")); ok {
			t.Error("expected From to return false")
		}
	})

	t.Run("returns false for nil", func(t *testing.T) {
		if _, ok := exception.From(nil); ok {
			t.Error("expected From to return false")
		}
	})
}

func TestWrap(t *testing.T) {
	t.Run("nil stays nil", func(t *testing.T) {
		if err := exception.Wrap(nil, exception.LayerApplication, "save lot"); err != nil {
			t.Fatalf("got %v, want nil", err)
		}
	})

	t.Run("unknown error becomes internal exception", func(t *testing.T) {
		cause := errors.New("connection refused")

		err := exception.Wrap(cause, exception.LayerInfrastructure, "save lot")

		got, ok := exception.From(err)
		if !ok {
			t.Fatal("expected an *Exception")
		}
		if got.Kind() != exception.KindInternal {
			t.Errorf("Kind: got %v, want %v", got.Kind(), exception.KindInternal)
		}
		if got.Layer() != exception.LayerInfrastructure {
			t.Errorf("Layer: got %v, want %v", got.Layer(), exception.LayerInfrastructure)
		}
		if got.TechnicalMessage() != "save lot" {
			t.Errorf("TechnicalMessage: got %q, want %q", got.TechnicalMessage(), "save lot")
		}
		if !errors.Is(err, cause) {
			t.Error("expected the original cause to be kept")
		}
	})

	t.Run("existing exception is not wrapped again", func(t *testing.T) {
		original := exception.NewValidation(exception.LayerDomain, "Stage out of order", "")
		wrapped := fmt.Errorf("register stage: %w", original)

		err := exception.Wrap(wrapped, exception.LayerApplication, "register stage")

		got, _ := exception.From(err)
		if got.Kind() != exception.KindValidation {
			t.Errorf("Kind changed: got %v, want %v", got.Kind(), exception.KindValidation)
		}
		if got.Layer() != exception.LayerDomain {
			t.Errorf("Layer changed: got %v, want %v", got.Layer(), exception.LayerDomain)
		}
	})
}
