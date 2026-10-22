package catalog_test

import (
	"errors"
	"strings"
	"testing"

	libcatalog "rutadeorigen/lib/catalog"
	"rutadeorigen/qr/internal/platform/catalog"
	"rutadeorigen/qr/internal/platform/exception"
)

var paramsByCode = map[exception.Code][]string{
	exception.CodeLotViewInvalidCode: {"min", "max"},
}

func newCatalog(t *testing.T) *libcatalog.Catalog {
	t.Helper()

	c, err := catalog.New()
	if err != nil {
		t.Fatalf("catalog.New: %v", err)
	}
	return c
}

func TestNewBuildsCatalog(t *testing.T) {
	c := newCatalog(t)

	if len(c.Codes()) == 0 {
		t.Fatal("catalog has no codes")
	}
}

func TestEveryCodeHasMessage(t *testing.T) {
	c := newCatalog(t)

	for _, code := range exception.AllCodes() {
		if !c.Has(string(code)) {
			t.Errorf("code %q is declared in codes.go but has no message", code)
		}
	}
}

func TestNoOrphanMessages(t *testing.T) {
	c := newCatalog(t)

	declared := make(map[string]bool)
	for _, code := range exception.AllCodes() {
		declared[string(code)] = true
	}

	for _, code := range c.Codes() {
		if !declared[code] {
			t.Errorf("message %q has no matching code in codes.go", code)
		}
	}
}

func TestCodesWithoutParamsResolveWithoutParams(t *testing.T) {
	c := newCatalog(t)

	for _, code := range exception.AllCodes() {
		if _, needsParams := paramsByCode[code]; needsParams {
			continue
		}

		t.Run(string(code), func(t *testing.T) {
			msg, err := c.Resolve(string(code), nil)
			if err != nil {
				t.Fatalf("Resolve: %v", err)
			}
			if strings.TrimSpace(msg) == "" {
				t.Error("message is empty")
			}
		})
	}
}

func TestCodesWithParamsFailWithoutThem(t *testing.T) {
	c := newCatalog(t)

	for code := range paramsByCode {
		t.Run(string(code), func(t *testing.T) {
			_, err := c.Resolve(string(code), nil)

			if !errors.Is(err, libcatalog.ErrMissingParam) {
				t.Errorf("error: got %v, want ErrMissingParam", err)
			}
		})
	}
}

func TestCodesWithParamsUseEveryDeclaredParam(t *testing.T) {
	c := newCatalog(t)

	for code, names := range paramsByCode {
		t.Run(string(code), func(t *testing.T) {
			params := make(map[string]any, len(names))
			for _, name := range names {
				params[name] = "<<" + name + ">>"
			}

			msg, err := c.Resolve(string(code), params)
			if err != nil {
				t.Fatalf("Resolve: %v", err)
			}

			for _, name := range names {
				if !strings.Contains(msg, "<<"+name+">>") {
					t.Errorf("message %q does not use parameter %q", msg, name)
				}
			}
		})
	}
}

func TestEveryCodeResolvesFromItsException(t *testing.T) {
	c := newCatalog(t)

	for _, code := range exception.AllCodes() {
		t.Run(string(code), func(t *testing.T) {
			ex := exception.New(code)

			if names, ok := paramsByCode[code]; ok {
				params := make(map[string]any, len(names))
				for _, name := range names {
					params[name] = 1
				}
				ex = ex.WithParams(params)
			}

			msg, err := c.Resolve(string(ex.Code()), ex.Params())
			if err != nil {
				t.Fatalf("Resolve: %v", err)
			}
			if strings.TrimSpace(msg) == "" {
				t.Error("message is empty")
			}
			if strings.ContainsAny(msg, "{}") {
				t.Errorf("message %q still has a placeholder", msg)
			}
		})
	}
}

func TestUnregisteredCodeResolvesToInternalMessage(t *testing.T) {
	c := newCatalog(t)

	ex := exception.New("NOT_A_REAL_CODE")
	got, err := c.Resolve(string(ex.Code()), ex.Params())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	want, err := c.Resolve(string(exception.CodeInternal), nil)
	if err != nil {
		t.Fatalf("Resolve INTERNAL: %v", err)
	}
	if got != want {
		t.Errorf("got %q, want the INTERNAL message %q", got, want)
	}
}
