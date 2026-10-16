package exception_test

import (
	"errors"
	"fmt"
	"testing"

	"rutadeorigen/traceability/internal/platform/exception"
)

func TestErrorWithoutCause(t *testing.T) {
	err := exception.New(exception.CodeLotInvalidVariety).WithTechnical("variety is empty")

	if got := err.Error(); got != "variety is empty" {
		t.Errorf("got %q, want %q", got, "variety is empty")
	}
}

func TestErrorWithCause(t *testing.T) {
	cause := errors.New("connection refused")
	err := exception.New(exception.CodeInternal).WithTechnical("save lot").WithCause(cause)

	if got := err.Error(); got != "save lot: connection refused" {
		t.Errorf("got %q, want %q", got, "save lot: connection refused")
	}
}

func TestUnwrap(t *testing.T) {
	cause := errors.New("connection refused")

	withCause := exception.New(exception.CodeInternal).WithCause(cause)
	if !errors.Is(withCause, cause) {
		t.Error("expected errors.Is to find the original cause")
	}

	withoutCause := exception.New(exception.CodeFarmNotFound)
	if withoutCause.Unwrap() != nil {
		t.Error("expected Unwrap to return nil when there is no cause")
	}
}

func TestGetters(t *testing.T) {
	err := exception.New(exception.CodeLotCodeAlreadyExists).WithTechnical("duplicate lot code")

	if err.TechnicalMessage() != "duplicate lot code" {
		t.Errorf("TechnicalMessage: got %q", err.TechnicalMessage())
	}
	if err.Kind() != exception.KindConflict {
		t.Errorf("Kind: got %v", err.Kind())
	}
	if err.Code() != exception.CodeLotCodeAlreadyExists {
		t.Errorf("Code: got %q", err.Code())
	}
}

func TestFrom(t *testing.T) {
	t.Run("finds exception directly", func(t *testing.T) {
		original := exception.New(exception.CodeLotInvalidVariety)

		got, ok := exception.From(original)
		if !ok || got != original {
			t.Error("expected From to return the same exception")
		}
	})

	t.Run("finds exception wrapped with fmt.Errorf", func(t *testing.T) {
		original := exception.New(exception.CodeLotInvalidVariety)
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
		if err := exception.Wrap(nil, "save lot"); err != nil {
			t.Fatalf("got %v, want nil", err)
		}
	})

	t.Run("unknown error becomes internal exception", func(t *testing.T) {
		cause := errors.New("connection refused")

		err := exception.Wrap(cause, "save lot")

		got, ok := exception.From(err)
		if !ok {
			t.Fatal("expected an *Exception")
		}
		if got.Kind() != exception.KindInternal || got.Code() != exception.CodeInternal {
			t.Errorf("got kind %v and code %q, want internal", got.Kind(), got.Code())
		}
		if got.TechnicalMessage() != "save lot" {
			t.Errorf("TechnicalMessage: got %q, want %q", got.TechnicalMessage(), "save lot")
		}
		if !errors.Is(err, cause) {
			t.Error("expected the original cause to be kept")
		}
	})

	t.Run("existing exception is not wrapped again", func(t *testing.T) {
		original := exception.New(exception.CodeLotStageOutOfOrder)
		wrapped := fmt.Errorf("register stage: %w", original)

		err := exception.Wrap(wrapped, "register stage")

		got, _ := exception.From(err)
		if got.Kind() != exception.KindConflict {
			t.Errorf("Kind changed: got %v, want %v", got.Kind(), exception.KindConflict)
		}
		if got.Code() != exception.CodeLotStageOutOfOrder {
			t.Errorf("Code changed: got %q, want %q", got.Code(), exception.CodeLotStageOutOfOrder)
		}
	})
}

func TestCode(t *testing.T) {
	t.Run("coded exception", func(t *testing.T) {
		ex := exception.New(exception.CodeFarmNotFound)

		if ex.Code() != exception.CodeFarmNotFound {
			t.Errorf("got %q, want %q", ex.Code(), exception.CodeFarmNotFound)
		}
	})

	t.Run("zero value has no code", func(t *testing.T) {
		if code := (&exception.Exception{}).Code(); code != "" {
			t.Errorf("got %q, want empty", code)
		}
	})
}

func TestParamsNeverNil(t *testing.T) {
	cases := map[string]*exception.Exception{
		"zero value":        {},
		"coded":             exception.New(exception.CodeLotVoided),
		"unregistered code": exception.New("NOT_A_REAL_CODE"),
	}

	for name, ex := range cases {
		t.Run(name, func(t *testing.T) {
			if ex.Params() == nil {
				t.Fatal("Params is nil, want empty map")
			}
			if len(ex.Params()) != 0 {
				t.Errorf("Params: got %v, want empty", ex.Params())
			}
		})
	}
}

func TestParamsReturnsCopy(t *testing.T) {
	ex := exception.New(exception.CodeLotInvalidCode).WithParams(map[string]any{"min": 3})

	got := ex.Params()
	got["min"] = 99
	got["extra"] = "x"

	again := ex.Params()
	if again["min"] != 3 {
		t.Errorf("min: got %v, want 3", again["min"])
	}
	if _, ok := again["extra"]; ok {
		t.Error("modifying the result of Params changed the exception")
	}
}

func TestHasCode(t *testing.T) {
	coded := exception.New(exception.CodeLotVoided)

	t.Run("direct", func(t *testing.T) {
		if !exception.HasCode(coded, exception.CodeLotVoided) {
			t.Error("expected true for the same code")
		}
	})

	t.Run("different code", func(t *testing.T) {
		if exception.HasCode(coded, exception.CodeLotHasEvents) {
			t.Error("expected false for a different code")
		}
	})

	t.Run("wrapped with fmt.Errorf", func(t *testing.T) {
		err := fmt.Errorf("register stage: %w", coded)

		if !exception.HasCode(err, exception.CodeLotVoided) {
			t.Error("expected true through a %w wrap")
		}
	})

	t.Run("kept by Wrap", func(t *testing.T) {
		err := exception.Wrap(coded, "register stage")

		if !exception.HasCode(err, exception.CodeLotVoided) {
			t.Error("Wrap must keep the original code")
		}
	})

	t.Run("exception with params", func(t *testing.T) {
		err := exception.New(exception.CodeLotInvalidCode).WithParams(map[string]any{"min": 3})

		if !exception.HasCode(err, exception.CodeLotInvalidCode) {
			t.Error("params must not affect the code")
		}
	})

	t.Run("plain error", func(t *testing.T) {
		if exception.HasCode(errors.New("boom"), exception.CodeLotVoided) {
			t.Error("expected false for a non-exception error")
		}
	})

	t.Run("nil error", func(t *testing.T) {
		if exception.HasCode(nil, exception.CodeLotVoided) {
			t.Error("expected false for nil")
		}
	})

	t.Run("exception without code", func(t *testing.T) {
		if exception.HasCode(&exception.Exception{}, exception.CodeLotVoided) {
			t.Error("expected false for an exception without code")
		}
	})
}

func TestIsThroughWrapping(t *testing.T) {
	err := fmt.Errorf("create lot: %w", exception.New(exception.CodeLotVoided))

	if !errors.Is(err, exception.New(exception.CodeLotVoided)) {
		t.Error("errors.Is must match a coded exception through %w")
	}
}
