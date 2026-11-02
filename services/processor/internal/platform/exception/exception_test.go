package exception_test

import (
	"errors"
	"testing"

	"rutadeorigen/processor/internal/platform/exception"
)

var errCause = errors.New("garage is down")

func TestNewUsesTheRegisteredKind(t *testing.T) {
	tests := []struct {
		code exception.Code
		kind exception.Kind
	}{
		{exception.CodeImageTooManyPixels, exception.KindValidation},
		{exception.CodeStorageUnavailable, exception.KindUnavailable},
		{exception.CodeImageProcessingFailed, exception.KindInternal},
	}

	for _, tt := range tests {
		ex := exception.New(tt.code)

		if ex.Code() != tt.code || ex.Kind() != tt.kind {
			t.Errorf("%s: got code %q and kind %q, want kind %q", tt.code, ex.Code(), ex.Kind(), tt.kind)
		}
		if ex.Error() != string(tt.code) {
			t.Errorf("%s: the technical message must default to the code, got %q", tt.code, ex.Error())
		}
	}
}

func TestNewWithAnUnregisteredCodeIsInternal(t *testing.T) {
	ex := exception.New("NOT_A_REAL_CODE")

	if ex.Code() != exception.CodeInternal || ex.Kind() != exception.KindInternal {
		t.Errorf("got code %q and kind %q, want INTERNAL", ex.Code(), ex.Kind())
	}
	if ex.TechnicalMessage() == "" {
		t.Error("the technical message must name the unregistered code")
	}
}

func TestWithParamsDoesNotMutateTheOriginal(t *testing.T) {
	base := exception.New(exception.CodeImageTooManyPixels)
	params := map[string]any{"max": 40}

	withParams := base.WithParams(params)
	params["max"] = 1

	if len(base.Params()) != 0 {
		t.Errorf("the original must keep no params, got %v", base.Params())
	}
	if withParams.Params()["max"] != 40 {
		t.Errorf("the params must be copied, got %v", withParams.Params())
	}

	returned := withParams.Params()
	returned["max"] = 2
	if withParams.Params()["max"] != 40 {
		t.Error("Params must return a copy")
	}
}

func TestWithTechnicalAndWithCause(t *testing.T) {
	base := exception.New(exception.CodeStorageUnavailable)

	ex := base.WithTechnical("  stat quarantine object ").WithCause(errCause)

	if ex.TechnicalMessage() != "stat quarantine object" {
		t.Errorf("technical message: got %q", ex.TechnicalMessage())
	}
	if ex.Error() != "stat quarantine object: garage is down" {
		t.Errorf("Error: got %q", ex.Error())
	}
	if !errors.Is(ex, errCause) {
		t.Error("the cause must be reachable with errors.Is")
	}
	if base.TechnicalMessage() != string(exception.CodeStorageUnavailable) {
		t.Error("the original must not change")
	}
}

func TestWrapKeepsExceptionsAndWrapsEverythingElse(t *testing.T) {
	if exception.Wrap(nil, "anything") != nil {
		t.Error("wrapping nil must return nil")
	}

	original := exception.New(exception.CodeImageCorrupt)
	if got := exception.Wrap(original, "decode"); got != error(original) {
		t.Errorf("an exception must be returned as it is, got %v", got)
	}

	wrapped := exception.Wrap(errCause, "put final image")
	ex, ok := exception.From(wrapped)
	if !ok || ex.Code() != exception.CodeInternal || ex.Kind() != exception.KindInternal {
		t.Fatalf("got %v, want an internal exception", wrapped)
	}
	if !errors.Is(wrapped, errCause) {
		t.Error("the original error must be kept as the cause")
	}
}

func TestFromAndHasCode(t *testing.T) {
	err := exception.Wrap(exception.New(exception.CodeFileMissing), "stat")

	if _, ok := exception.From(errCause); ok {
		t.Error("a plain error is not an exception")
	}
	if !exception.HasCode(err, exception.CodeFileMissing) {
		t.Error("HasCode must find the code")
	}
	if exception.HasCode(err, exception.CodeFileTooLarge) || exception.HasCode(errCause, exception.CodeFileMissing) || exception.HasCode(nil, exception.CodeFileMissing) {
		t.Error("HasCode must be false for other codes, plain errors and nil")
	}
}

func TestIsComparesByCode(t *testing.T) {
	err := exception.New(exception.CodeImageCorrupt).WithTechnical("decode jpeg")

	if !errors.Is(err, exception.New(exception.CodeImageCorrupt)) {
		t.Error("two exceptions with the same code must match")
	}
	if errors.Is(err, exception.New(exception.CodeImageUnsupportedFormat)) {
		t.Error("two exceptions with different codes must not match")
	}
}
