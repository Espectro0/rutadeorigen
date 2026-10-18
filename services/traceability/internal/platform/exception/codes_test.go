package exception_test

import (
	"regexp"
	"strings"
	"testing"

	"rutadeorigen/traceability/internal/platform/exception"
)

const servicePrefix = "TRACEABILITY_"

var genericCodes = map[exception.Code]bool{
	exception.CodeInternal:           true,
	exception.CodeInvalidRequestBody: true,
	exception.CodeUnauthorized:       true,
	exception.CodeForbidden:          true,
	exception.CodeInvalidCursor:      true,
	exception.CodeInvalidLimit:       true,
}

var codeFormat = regexp.MustCompile(`^[A-Z][A-Z0-9]*(_[A-Z0-9]+)*$`)

func TestAllCodesHaveKind(t *testing.T) {
	for _, code := range exception.AllCodes() {
		kind, ok := exception.KindOf(code)
		if !ok {
			t.Errorf("code %q is listed but KindOf does not know it", code)
		}
		if kind == "" {
			t.Errorf("code %q has an empty kind", code)
		}
	}
}

func TestKindOfUnknownCode(t *testing.T) {
	if _, ok := exception.KindOf("NOT_A_REAL_CODE"); ok {
		t.Error("KindOf reported an unregistered code as known")
	}
}

func TestAllCodesHaveValidFormat(t *testing.T) {
	for _, code := range exception.AllCodes() {
		if !codeFormat.MatchString(string(code)) {
			t.Errorf("code %q must be UPPER_SNAKE_CASE", code)
		}
	}
}

func TestAllCodesFollowPrefixConvention(t *testing.T) {
	for _, code := range exception.AllCodes() {
		hasPrefix := strings.HasPrefix(string(code), servicePrefix)

		if genericCodes[code] && hasPrefix {
			t.Errorf("generic code %q must not carry the %s prefix", code, servicePrefix)
		}
		if !genericCodes[code] && !hasPrefix {
			t.Errorf("service code %q must start with %s", code, servicePrefix)
		}
	}
}

func TestGenericCodesAreRegistered(t *testing.T) {
	registered := make(map[exception.Code]bool)
	for _, code := range exception.AllCodes() {
		registered[code] = true
	}

	for code := range genericCodes {
		if !registered[code] {
			t.Errorf("generic code %q is not registered", code)
		}
	}
}

func TestEveryCodeBuildsMatchingException(t *testing.T) {
	for _, code := range exception.AllCodes() {
		ex := exception.New(code)
		wantKind, _ := exception.KindOf(code)

		if ex.Code() != code {
			t.Errorf("New(%q).Code() = %q", code, ex.Code())
		}
		if ex.Kind() != wantKind {
			t.Errorf("New(%q).Kind() = %q, want %q", code, ex.Kind(), wantKind)
		}
	}
}
