package exception_test

import (
	"errors"
	"testing"

	"rutadeorigen/qr/internal/platform/exception"
)

func TestNewTakesKindFromRegistry(t *testing.T) {
	ex := exception.New(exception.CodeLotViewImageNotAvailable)

	if ex.Code() != exception.CodeLotViewImageNotAvailable {
		t.Errorf("Code: got %q, want %q", ex.Code(), exception.CodeLotViewImageNotAvailable)
	}
	if ex.Kind() != exception.KindConflict {
		t.Errorf("Kind: got %v, want %v", ex.Kind(), exception.KindConflict)
	}
	if ex.TechnicalMessage() != string(exception.CodeLotViewImageNotAvailable) {
		t.Errorf("TechnicalMessage: got %q, want %q", ex.TechnicalMessage(), exception.CodeLotViewImageNotAvailable)
	}
	if ex.Unwrap() != nil {
		t.Errorf("Unwrap: got %v, want nil", ex.Unwrap())
	}
}

func TestNewUnregisteredCodeFallsBackToInternal(t *testing.T) {
	ex := exception.New("NOT_A_REAL_CODE")

	if ex.Code() != exception.CodeInternal {
		t.Errorf("Code: got %q, want %q", ex.Code(), exception.CodeInternal)
	}
	if ex.Kind() != exception.KindInternal {
		t.Errorf("Kind: got %v, want %v", ex.Kind(), exception.KindInternal)
	}
	want := "unregistered error code: NOT_A_REAL_CODE"
	if ex.TechnicalMessage() != want {
		t.Errorf("TechnicalMessage: got %q, want %q", ex.TechnicalMessage(), want)
	}
}

func TestWithParamsReturnsCopy(t *testing.T) {
	original := exception.New(exception.CodeLotViewInvalidCode)
	withParams := original.WithParams(map[string]any{"min": 3, "max": 30})

	if withParams == original {
		t.Fatal("WithParams returned the same pointer")
	}
	if len(original.Params()) != 0 {
		t.Errorf("original params changed: got %v", original.Params())
	}
	if withParams.Params()["min"] != 3 || withParams.Params()["max"] != 30 {
		t.Errorf("params: got %v", withParams.Params())
	}
	if withParams.Code() != original.Code() || withParams.Kind() != original.Kind() {
		t.Errorf("code or kind changed: got %q/%v, want %q/%v",
			withParams.Code(), withParams.Kind(), original.Code(), original.Kind())
	}
}

func TestWithParamsDoesNotAliasInput(t *testing.T) {
	input := map[string]any{"min": 3}
	ex := exception.New(exception.CodeLotViewInvalidCode).WithParams(input)

	input["min"] = 99
	input["extra"] = "x"

	got := ex.Params()
	if got["min"] != 3 {
		t.Errorf("min: got %v, want 3", got["min"])
	}
	if _, ok := got["extra"]; ok {
		t.Error("exception picked up a key added to the caller's map")
	}
}

func TestWithParamsNilGivesEmptyMap(t *testing.T) {
	ex := exception.New(exception.CodeLotViewInvalidCode).WithParams(nil)

	if ex.Params() == nil {
		t.Fatal("Params is nil, want empty map")
	}
	if len(ex.Params()) != 0 {
		t.Errorf("Params: got %v, want empty", ex.Params())
	}
}

func TestWithTechnical(t *testing.T) {
	t.Run("returns a copy with the new message", func(t *testing.T) {
		original := exception.New(exception.CodeInternal)

		got := original.WithTechnical("  cannot save a default lot  ")

		if got == original {
			t.Fatal("WithTechnical returned the same pointer")
		}
		if got.TechnicalMessage() != "cannot save a default lot" {
			t.Errorf("TechnicalMessage: got %q, want the trimmed message", got.TechnicalMessage())
		}
		if original.TechnicalMessage() != string(exception.CodeInternal) {
			t.Errorf("original changed: got %q", original.TechnicalMessage())
		}
	})

	t.Run("keeps code, kind and params", func(t *testing.T) {
		original := exception.New(exception.CodeLotViewInvalidCode).WithParams(map[string]any{"min": 3})

		got := original.WithTechnical("code too short")

		if got.Code() != original.Code() || got.Kind() != original.Kind() || got.Params()["min"] != 3 {
			t.Errorf("code, kind or params changed: %q/%v/%v", got.Code(), got.Kind(), got.Params())
		}
	})
}

func TestWithCause(t *testing.T) {
	cause := errors.New("connection refused")

	t.Run("returns a copy that keeps the cause", func(t *testing.T) {
		original := exception.New(exception.CodeInternal)

		got := original.WithCause(cause)

		if got == original {
			t.Fatal("WithCause returned the same pointer")
		}
		if !errors.Is(got, cause) {
			t.Error("expected the cause to be reachable with errors.Is")
		}
		if original.Unwrap() != nil {
			t.Error("original must not get the cause")
		}
	})

	t.Run("keeps matching by code", func(t *testing.T) {
		got := exception.New(exception.CodeInternal).WithCause(cause)

		if !errors.Is(got, exception.New(exception.CodeInternal)) {
			t.Error("an exception with a cause must still match its code")
		}
	})
}

func TestIsMatchesByCodeIgnoringParams(t *testing.T) {
	a := exception.New(exception.CodeLotViewInvalidCode).WithParams(map[string]any{"min": 3})
	b := exception.New(exception.CodeLotViewInvalidCode).WithParams(map[string]any{"min": 5})
	other := exception.New(exception.CodeLotViewImageNotAvailable)

	if !errors.Is(a, b) {
		t.Error("exceptions with the same code must match")
	}
	if errors.Is(a, other) {
		t.Error("exceptions with different codes must not match")
	}
}

func TestIsIgnoresExceptionsWithoutCode(t *testing.T) {
	a := &exception.Exception{}
	b := &exception.Exception{}

	if errors.Is(a, b) {
		t.Error("exceptions without code must not match each other")
	}
}

func TestIsDoesNotMatchPlainError(t *testing.T) {
	ex := exception.New(exception.CodeLotViewImageNotAvailable)

	if errors.Is(ex, errors.New("lot is voided")) {
		t.Error("an exception must not match an unrelated error")
	}
}
