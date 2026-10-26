package session

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func findCookie(t *testing.T, rec *httptest.ResponseRecorder, name string) *http.Cookie {
	t.Helper()
	for _, c := range rec.Result().Cookies() {
		if c.Name == name {
			return c
		}
	}
	t.Fatalf("cookie %q not found in %v", name, rec.Result().Cookies())
	return nil
}

func TestCookieSetAppliesDefaults(t *testing.T) {
	rec := httptest.NewRecorder()

	if err := NewCookie(CookieConfig{}).Set(rec, "sealed"); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	c := findCookie(t, rec, defaultCookieName)
	if c.Value != "sealed" {
		t.Errorf("Value = %q, want %q", c.Value, "sealed")
	}
	if c.Path != defaultCookiePath {
		t.Errorf("Path = %q, want %q", c.Path, defaultCookiePath)
	}
	if c.MaxAge != int(defaultCookieMaxAge.Seconds()) {
		t.Errorf("MaxAge = %d, want %d", c.MaxAge, int(defaultCookieMaxAge.Seconds()))
	}
	if c.SameSite != http.SameSiteLaxMode {
		t.Errorf("SameSite = %v, want Lax", c.SameSite)
	}
	if !c.HttpOnly {
		t.Error("HttpOnly = false, want true")
	}
	if c.Secure {
		t.Error("Secure = true, want false")
	}
}

func TestCookieSetKeepsCustomConfig(t *testing.T) {
	rec := httptest.NewRecorder()
	cookie := NewCookie(CookieConfig{
		Name:     "custom",
		Path:     "/api",
		Domain:   "example.com",
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   time.Hour,
	})

	if err := cookie.Set(rec, "sealed"); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	c := findCookie(t, rec, "custom")
	if c.Path != "/api" {
		t.Errorf("Path = %q, want %q", c.Path, "/api")
	}
	if c.Domain != "example.com" {
		t.Errorf("Domain = %q, want %q", c.Domain, "example.com")
	}
	if c.MaxAge != 3600 {
		t.Errorf("MaxAge = %d, want 3600", c.MaxAge)
	}
	if c.SameSite != http.SameSiteStrictMode {
		t.Errorf("SameSite = %v, want Strict", c.SameSite)
	}
	if !c.Secure {
		t.Error("Secure = false, want true")
	}
	if !c.HttpOnly {
		t.Error("HttpOnly = false, want true")
	}
}

func TestCookieSetRejectsOversizedValue(t *testing.T) {
	rec := httptest.NewRecorder()

	err := NewCookie(CookieConfig{}).Set(rec, strings.Repeat("a", maxCookieValueBytes+1))

	if !errors.Is(err, ErrCookieTooLarge) {
		t.Fatalf("Set() error = %v, want %v", err, ErrCookieTooLarge)
	}
	if got := rec.Header().Get("Set-Cookie"); got != "" {
		t.Errorf("Set-Cookie = %q, want empty", got)
	}
}

func TestCookieSetAcceptsValueAtLimit(t *testing.T) {
	rec := httptest.NewRecorder()

	err := NewCookie(CookieConfig{}).Set(rec, strings.Repeat("a", maxCookieValueBytes))

	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	if got := findCookie(t, rec, defaultCookieName); len(got.Value) != maxCookieValueBytes {
		t.Errorf("len(Value) = %d, want %d", len(got.Value), maxCookieValueBytes)
	}
}

func TestCookieRead(t *testing.T) {
	tests := []struct {
		name   string
		cookie *http.Cookie
		want   string
		wantOK bool
	}{
		{name: "present", cookie: &http.Cookie{Name: defaultCookieName, Value: "sealed"}, want: "sealed", wantOK: true},
		{name: "empty value", cookie: &http.Cookie{Name: defaultCookieName, Value: ""}},
		{name: "other cookie", cookie: &http.Cookie{Name: "other", Value: "sealed"}},
		{name: "absent"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.cookie != nil {
				req.AddCookie(tt.cookie)
			}

			got, ok := NewCookie(CookieConfig{}).Read(req)

			if got != tt.want || ok != tt.wantOK {
				t.Errorf("Read() = (%q, %v), want (%q, %v)", got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

func TestCookieClearExpiresWithSameScope(t *testing.T) {
	rec := httptest.NewRecorder()
	cookie := NewCookie(CookieConfig{Path: "/api", Domain: "example.com", Secure: true})

	cookie.Clear(rec)

	c := findCookie(t, rec, defaultCookieName)
	if c.Value != "" {
		t.Errorf("Value = %q, want empty", c.Value)
	}
	if c.MaxAge >= 0 {
		t.Errorf("MaxAge = %d, want negative", c.MaxAge)
	}
	if c.Path != "/api" {
		t.Errorf("Path = %q, want %q", c.Path, "/api")
	}
	if c.Domain != "example.com" {
		t.Errorf("Domain = %q, want %q", c.Domain, "example.com")
	}
	if !c.HttpOnly || !c.Secure {
		t.Errorf("HttpOnly = %v, Secure = %v, want both true", c.HttpOnly, c.Secure)
	}
}

func TestCookieRoundTrip(t *testing.T) {
	cookie := NewCookie(CookieConfig{})
	rec := httptest.NewRecorder()
	if err := cookie.Set(rec, "sealed"); err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	for _, c := range rec.Result().Cookies() {
		req.AddCookie(c)
	}

	got, ok := cookie.Read(req)

	if !ok || got != "sealed" {
		t.Errorf("Read() = (%q, %v), want (%q, true)", got, ok, "sealed")
	}
}
