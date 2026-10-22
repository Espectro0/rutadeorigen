package redirect_test

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"rutadeorigen/qr/internal/domain/redirect"
	"rutadeorigen/qr/internal/platform/exception"
)

func newRule(t *testing.T) *redirect.Rule {
	t.Helper()
	r, err := redirect.New("/l/", "/lotes/{id}", 301)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return r
}

func TestNewValid(t *testing.T) {
	r, err := redirect.New("  /l/  ", "  https://front.rutadeorigen.co/lotes/{id}  ", 302)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if r.ID() == uuid.Nil || r.CreatedAt().IsZero() {
		t.Error("expected generated ID and creation date")
	}
	if r.SourcePrefix() != "/l/" || r.TargetTemplate() != "https://front.rutadeorigen.co/lotes/{id}" {
		t.Errorf("expected trimmed values, got %q and %q", r.SourcePrefix(), r.TargetTemplate())
	}
	if r.StatusCode() != 302 || !r.Active() || r.IsDefault() {
		t.Error("unexpected status code, active flag or default state")
	}
}

func TestNewInvalid(t *testing.T) {
	tests := []struct {
		name     string
		prefix   string
		template string
		status   int
	}{
		{"empty prefix", "", "/lotes/{id}", 301},
		{"prefix is only a slash", "/", "/lotes/{id}", 301},
		{"prefix without leading slash", "l/", "/lotes/{id}", 301},
		{"prefix without trailing slash", "/l", "/lotes/{id}", 301},
		{"empty template", "/l/", "", 301},
		{"template without placeholder", "/l/", "/lotes", 301},
		{"template with an unsafe scheme", "/l/", "javascript:{id}", 301},
		{"relative template without slash", "/l/", "lotes/{id}", 301},
		{"unsupported status code", "/l/", "/lotes/{id}", 200},
		{"zero status code", "/l/", "/lotes/{id}", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := redirect.New(tt.prefix, tt.template, tt.status)
			if !exception.HasCode(err, exception.CodeRedirectInvalidRule) {
				t.Errorf("got %v, want invalid rule", err)
			}
			if !r.IsDefault() {
				t.Error("expected default rule on error")
			}
		})
	}
}

func TestNewAcceptsEveryAllowedStatusCode(t *testing.T) {
	for _, status := range []int{301, 302, 308} {
		if _, err := redirect.New("/l/", "/lotes/{id}", status); err != nil {
			t.Errorf("status %d: unexpected error: %v", status, err)
		}
	}
}

func TestDefault(t *testing.T) {
	r := redirect.Default()

	if !r.IsDefault() || r.Active() {
		t.Error("Default must be default and inactive")
	}
	if r.SourcePrefix() != "" || r.TargetTemplate() != "" || r.StatusCode() != 0 {
		t.Error("unexpected default values")
	}
	if redirect.Default() == r {
		t.Error("Default must return a new instance each time")
	}
}

func TestIsDefaultOnNil(t *testing.T) {
	var r *redirect.Rule

	if !r.IsDefault() {
		t.Error("a nil rule must be default")
	}
}

func TestRestoreKeepsEverything(t *testing.T) {
	p := redirect.RestoreParams{
		ID:             uuid.New(),
		SourcePrefix:   "/l/",
		TargetTemplate: "/lotes/{id}",
		StatusCode:     308,
		Active:         false,
		CreatedAt:      time.Now().UTC(),
	}

	r := redirect.Restore(p)

	if r.ID() != p.ID || r.SourcePrefix() != p.SourcePrefix || r.TargetTemplate() != p.TargetTemplate {
		t.Error("fields were not restored")
	}
	if r.StatusCode() != p.StatusCode || r.Active() != p.Active || !r.CreatedAt().Equal(p.CreatedAt) {
		t.Error("fields were not restored")
	}
}

func TestMatches(t *testing.T) {
	r := newRule(t)

	if !r.Matches("/l/" + uuid.NewString()) {
		t.Error("expected a match for the prefix")
	}
	if r.Matches("/x/" + uuid.NewString()) {
		t.Error("did not expect a match for another prefix")
	}
	if r.Matches("l/" + uuid.NewString()) {
		t.Error("did not expect a match without the leading slash")
	}

	inactive := redirect.Restore(redirect.RestoreParams{
		ID: uuid.New(), SourcePrefix: "/l/", TargetTemplate: "/lotes/{id}", StatusCode: 301, Active: false,
	})
	if inactive.Matches("/l/" + uuid.NewString()) {
		t.Error("an inactive rule must not match")
	}
}

func TestResolve(t *testing.T) {
	lotID := uuid.New()

	t.Run("relative target", func(t *testing.T) {
		got, err := newRule(t).Resolve("/l/" + lotID.String())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.LotID != lotID || got.Location != "/lotes/"+lotID.String() || got.StatusCode != 301 {
			t.Errorf("unexpected resolution: %+v", got)
		}
	})

	t.Run("absolute target", func(t *testing.T) {
		r, err := redirect.New("/l/", "https://front.rutadeorigen.co/lotes/{id}", 302)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got, err := r.Resolve("/l/" + lotID.String())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Location != "https://front.rutadeorigen.co/lotes/"+lotID.String() || got.StatusCode != 302 {
			t.Errorf("unexpected resolution: %+v", got)
		}
	})

	t.Run("template with the placeholder twice", func(t *testing.T) {
		r, err := redirect.New("/l/", "/lotes/{id}?ref={id}", 301)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got, err := r.Resolve("/l/" + lotID.String())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if want := "/lotes/" + lotID.String() + "?ref=" + lotID.String(); got.Location != want {
			t.Errorf("Location: got %q, want %q", got.Location, want)
		}
	})
}

func TestResolveNoMatch(t *testing.T) {
	_, err := newRule(t).Resolve("/x/" + uuid.NewString())

	if !exception.HasCode(err, exception.CodeRedirectNotFound) {
		t.Errorf("got %v, want redirect not found", err)
	}
}

func TestResolveInactive(t *testing.T) {
	r := redirect.Restore(redirect.RestoreParams{
		ID: uuid.New(), SourcePrefix: "/l/", TargetTemplate: "/lotes/{id}", StatusCode: 301, Active: false,
	})

	_, err := r.Resolve("/l/" + uuid.NewString())

	if !exception.HasCode(err, exception.CodeRedirectNotFound) {
		t.Errorf("got %v, want redirect not found", err)
	}
}

func TestResolveInvalidLotID(t *testing.T) {
	for _, path := range []string{"/l/", "/l/not-a-uuid", "/l/" + uuid.Nil.String(), "/l/" + uuid.NewString() + "/extra"} {
		t.Run(path, func(t *testing.T) {
			got, err := newRule(t).Resolve(path)

			if !exception.HasCode(err, exception.CodeLotViewInvalidID) {
				t.Errorf("got %v, want invalid lot ID", err)
			}
			if got.LotID != uuid.Nil || got.Location != "" || got.StatusCode != 0 {
				t.Errorf("expected an empty resolution, got %+v", got)
			}
		})
	}
}
