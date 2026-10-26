package session

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func requestWithState(value string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/auth/callback", nil)
	if value != "" {
		req.AddCookie(&http.Cookie{Name: stateCookieName, Value: value})
	}
	return req
}

func TestStateIssueSetsCookieAndReturnsValue(t *testing.T) {
	rec := httptest.NewRecorder()

	value, err := NewState(StateConfig{Domain: "example.com", Secure: true}).Issue(rec)

	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}
	if value == "" {
		t.Fatal("Issue() returned empty state")
	}
	c := findCookie(t, rec, stateCookieName)
	if c.Value != value {
		t.Errorf("cookie Value = %q, want %q", c.Value, value)
	}
	if c.Path != stateCookiePath {
		t.Errorf("Path = %q, want %q", c.Path, stateCookiePath)
	}
	if c.Domain != "example.com" {
		t.Errorf("Domain = %q, want %q", c.Domain, "example.com")
	}
	if c.MaxAge != int(stateCookieTTL.Seconds()) {
		t.Errorf("MaxAge = %d, want %d", c.MaxAge, int(stateCookieTTL.Seconds()))
	}
	if c.SameSite != http.SameSiteLaxMode {
		t.Errorf("SameSite = %v, want Lax", c.SameSite)
	}
	if !c.HttpOnly || !c.Secure {
		t.Errorf("HttpOnly = %v, Secure = %v, want both true", c.HttpOnly, c.Secure)
	}
}

func TestStateIssueReturnsDistinctValues(t *testing.T) {
	state := NewState(StateConfig{})

	first, err := state.Issue(httptest.NewRecorder())
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}
	second, err := state.Issue(httptest.NewRecorder())
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}

	if first == second {
		t.Errorf("Issue() returned the same state twice: %q", first)
	}
	if len(first) != 43 {
		t.Errorf("len(state) = %d, want 43", len(first))
	}
}

func TestStateVerify(t *testing.T) {
	tests := []struct {
		name     string
		cookie   string
		received string
		wantErr  error
	}{
		{name: "matching", cookie: "abc", received: "abc"},
		{name: "different", cookie: "abc", received: "abd", wantErr: ErrInvalidState},
		{name: "missing cookie", cookie: "", received: "abc", wantErr: ErrInvalidState},
		{name: "empty received", cookie: "abc", received: "", wantErr: ErrInvalidState},
		{name: "both empty", wantErr: ErrInvalidState},
		{name: "prefix only", cookie: "abcdef", received: "abc", wantErr: ErrInvalidState},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()

			err := NewState(StateConfig{}).Verify(rec, requestWithState(tt.cookie), tt.received)

			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Verify() error = %v, want %v", err, tt.wantErr)
			}
			c := findCookie(t, rec, stateCookieName)
			if c.MaxAge >= 0 || c.Value != "" {
				t.Errorf("state cookie not cleared: MaxAge = %d, Value = %q", c.MaxAge, c.Value)
			}
		})
	}
}

func TestStateVerifyClearsWithSameScope(t *testing.T) {
	rec := httptest.NewRecorder()

	_ = NewState(StateConfig{Domain: "example.com", Secure: true}).Verify(rec, requestWithState("abc"), "abc")

	c := findCookie(t, rec, stateCookieName)
	if c.Path != stateCookiePath || c.Domain != "example.com" {
		t.Errorf("Path = %q, Domain = %q, want %q and %q", c.Path, c.Domain, stateCookiePath, "example.com")
	}
	if !c.HttpOnly || !c.Secure {
		t.Errorf("HttpOnly = %v, Secure = %v, want both true", c.HttpOnly, c.Secure)
	}
}

func TestStateIssueThenVerify(t *testing.T) {
	state := NewState(StateConfig{})
	rec := httptest.NewRecorder()
	value, err := state.Issue(rec)
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/auth/callback", nil)
	for _, c := range rec.Result().Cookies() {
		req.AddCookie(c)
	}

	if err := state.Verify(httptest.NewRecorder(), req, value); err != nil {
		t.Errorf("Verify() error = %v, want nil", err)
	}
}
