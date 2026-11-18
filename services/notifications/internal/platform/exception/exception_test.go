package exception_test

import (
	"errors"
	"testing"

	"rutadeorigen/notifications/internal/platform/exception"
)

var errCause = errors.New("resend is down")

func TestNewUsesTheRegisteredKind(t *testing.T) {
	tests := []struct {
		code exception.Code
		kind exception.Kind
	}{
		{exception.CodeInvalidRecipient, exception.KindValidation},
		{exception.CodeEmailUnavailable, exception.KindUnavailable},
		{exception.CodeEmailRenderFailed, exception.KindInternal},
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
	base := exception.New(exception.CodeInvalidRecipient)
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
	base := exception.New(exception.CodeEmailUnavailable)

	ex := base.WithTechnical("  send email ").WithCause(errCause)

	if ex.TechnicalMessage() != "send email" {
		t.Errorf("technical message: got %q", ex.TechnicalMessage())
	}
	if ex.Error() != "send email: resend is down" {
		t.Errorf("Error: got %q", ex.Error())
	}
	if !errors.Is(ex, errCause) {
		t.Error("the cause must be reachable with errors.Is")
	}
	if base.TechnicalMessage() != string(exception.CodeEmailUnavailable) {
		t.Error("the original must not change")
	}
}

func TestWrapKeepsExceptionsAndWrapsEverythingElse(t *testing.T) {
	if exception.Wrap(nil, "anything") != nil {
		t.Error("wrapping nil must return nil")
	}

	original := exception.New(exception.CodeEmailRejected)
	if got := exception.Wrap(original, "decode"); !errors.Is(got, original) {
		t.Errorf("an exception must be returned as it is, got %v", got)
	}

	wrapped := exception.Wrap(errCause, "send email")
	ex, ok := exception.From(wrapped)
	if !ok || ex.Code() != exception.CodeInternal || ex.Kind() != exception.KindInternal {
		t.Fatalf("got %v, want an internal exception", wrapped)
	}
	if !errors.Is(wrapped, errCause) {
		t.Error("the original error must be kept as the cause")
	}
}

func TestFromAndHasCode(t *testing.T) {
	err := exception.Wrap(exception.New(exception.CodeEmailRejected), "send")

	if _, ok := exception.From(errCause); ok {
		t.Error("a plain error is not an exception")
	}
	if !exception.HasCode(err, exception.CodeEmailRejected) {
		t.Error("HasCode must find the code")
	}
	if exception.HasCode(err, exception.CodeEmailUnavailable) || exception.HasCode(errCause, exception.CodeEmailRejected) || exception.HasCode(nil, exception.CodeEmailRejected) {
		t.Error("HasCode must be false for other codes, plain errors and nil")
	}
}

func TestIsComparesByCode(t *testing.T) {
	err := exception.New(exception.CodeEmailRejected).WithTechnical("resend 422")

	if !errors.Is(err, exception.New(exception.CodeEmailRejected)) {
		t.Error("two exceptions with the same code must match")
	}
	if errors.Is(err, exception.New(exception.CodeEmailUnavailable)) {
		t.Error("two exceptions with different codes must not match")
	}
}
