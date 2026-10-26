package session

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type sessionFake struct {
	authUserID   string
	authErr      error
	refSealed    string
	refUserID    string
	refErr       error
	refreshCalls int
}

func (s *sessionFake) Authenticate() (string, error) {
	return s.authUserID, s.authErr
}

func (s *sessionFake) Refresh(_ context.Context) (string, string, error) {
	s.refreshCalls++
	return s.refSealed, s.refUserID, s.refErr
}

type factoryFake struct {
	session *sessionFake
	calls   int
	gotSeal string
}

func (f *factoryFake) build(sealed string) Session {
	f.calls++
	f.gotSeal = sealed
	return f.session
}

func authRequest(sealed string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if sealed != "" {
		req.AddCookie(&http.Cookie{Name: defaultCookieName, Value: sealed})
	}
	return req
}

func newTestAuthenticator(s *sessionFake) (*Authenticator, *factoryFake) {
	factory := &factoryFake{session: s}
	return NewAuthenticator(factory.build, NewCookie(CookieConfig{})), factory
}

func TestAuthenticatorWithoutCookie(t *testing.T) {
	a, factory := newTestAuthenticator(&sessionFake{})
	rec := httptest.NewRecorder()

	userID, err := a.Authenticate(rec, authRequest(""))

	if !errors.Is(err, ErrUnauthenticated) {
		t.Fatalf("error = %v, want %v", err, ErrUnauthenticated)
	}
	if userID != "" {
		t.Errorf("userID = %q, want empty", userID)
	}
	if factory.calls != 0 {
		t.Errorf("factory calls = %d, want 0", factory.calls)
	}
	if len(rec.Result().Cookies()) != 0 {
		t.Errorf("cookies = %v, want none", rec.Result().Cookies())
	}
}

func TestAuthenticatorValidSession(t *testing.T) {
	s := &sessionFake{authUserID: "user_1"}
	a, factory := newTestAuthenticator(s)
	rec := httptest.NewRecorder()

	userID, err := a.Authenticate(rec, authRequest("sealed-1"))

	if err != nil {
		t.Fatalf("error = %v, want nil", err)
	}
	if userID != "user_1" {
		t.Errorf("userID = %q, want %q", userID, "user_1")
	}
	if factory.gotSeal != "sealed-1" {
		t.Errorf("factory received %q, want %q", factory.gotSeal, "sealed-1")
	}
	if s.refreshCalls != 0 {
		t.Errorf("refresh calls = %d, want 0", s.refreshCalls)
	}
	if len(rec.Result().Cookies()) != 0 {
		t.Errorf("cookies = %v, want none", rec.Result().Cookies())
	}
}

func TestAuthenticatorRefreshesExpiredSession(t *testing.T) {
	s := &sessionFake{
		authErr:   ErrUnauthenticated,
		refSealed: "sealed-2",
		refUserID: "user_1",
	}
	a, _ := newTestAuthenticator(s)
	rec := httptest.NewRecorder()

	userID, err := a.Authenticate(rec, authRequest("sealed-1"))

	if err != nil {
		t.Fatalf("error = %v, want nil", err)
	}
	if userID != "user_1" {
		t.Errorf("userID = %q, want %q", userID, "user_1")
	}
	if s.refreshCalls != 1 {
		t.Errorf("refresh calls = %d, want 1", s.refreshCalls)
	}
	c := findCookie(t, rec, defaultCookieName)
	if c.Value != "sealed-2" {
		t.Errorf("cookie Value = %q, want %q", c.Value, "sealed-2")
	}
	if c.MaxAge <= 0 {
		t.Errorf("cookie MaxAge = %d, want positive", c.MaxAge)
	}
}

func TestAuthenticatorClearsCookieWhenRefreshIsRejected(t *testing.T) {
	s := &sessionFake{authErr: ErrUnauthenticated, refErr: ErrUnauthenticated}
	a, _ := newTestAuthenticator(s)
	rec := httptest.NewRecorder()

	userID, err := a.Authenticate(rec, authRequest("sealed-1"))

	if !errors.Is(err, ErrUnauthenticated) {
		t.Fatalf("error = %v, want %v", err, ErrUnauthenticated)
	}
	if userID != "" {
		t.Errorf("userID = %q, want empty", userID)
	}
	c := findCookie(t, rec, defaultCookieName)
	if c.MaxAge >= 0 || c.Value != "" {
		t.Errorf("cookie not cleared: MaxAge = %d, Value = %q", c.MaxAge, c.Value)
	}
}

func TestAuthenticatorKeepsCookieWhenRefreshIsUnavailable(t *testing.T) {
	cause := fmt.Errorf("%w: connection refused", ErrAuthUnavailable)
	s := &sessionFake{authErr: ErrUnauthenticated, refErr: cause}
	a, _ := newTestAuthenticator(s)
	rec := httptest.NewRecorder()

	_, err := a.Authenticate(rec, authRequest("sealed-1"))

	if !errors.Is(err, ErrAuthUnavailable) {
		t.Fatalf("error = %v, want %v", err, ErrAuthUnavailable)
	}
	if len(rec.Result().Cookies()) != 0 {
		t.Errorf("cookies = %v, want none", rec.Result().Cookies())
	}
}

func TestAuthenticatorDoesNotRefreshWhenValidationIsUnavailable(t *testing.T) {
	s := &sessionFake{authErr: fmt.Errorf("%w: jwks down", ErrAuthUnavailable)}
	a, _ := newTestAuthenticator(s)
	rec := httptest.NewRecorder()

	_, err := a.Authenticate(rec, authRequest("sealed-1"))

	if !errors.Is(err, ErrAuthUnavailable) {
		t.Fatalf("error = %v, want %v", err, ErrAuthUnavailable)
	}
	if s.refreshCalls != 0 {
		t.Errorf("refresh calls = %d, want 0", s.refreshCalls)
	}
	if len(rec.Result().Cookies()) != 0 {
		t.Errorf("cookies = %v, want none", rec.Result().Cookies())
	}
}

func TestAuthenticatorFailsWhenRefreshedCookieIsTooLarge(t *testing.T) {
	s := &sessionFake{
		authErr:   ErrUnauthenticated,
		refSealed: strings.Repeat("a", maxCookieValueBytes+1),
		refUserID: "user_1",
	}
	a, _ := newTestAuthenticator(s)
	rec := httptest.NewRecorder()

	userID, err := a.Authenticate(rec, authRequest("sealed-1"))

	if !errors.Is(err, ErrCookieTooLarge) {
		t.Fatalf("error = %v, want %v", err, ErrCookieTooLarge)
	}
	if userID != "" {
		t.Errorf("userID = %q, want empty", userID)
	}
	if len(rec.Result().Cookies()) != 0 {
		t.Errorf("cookies = %v, want none", rec.Result().Cookies())
	}
}

func TestAuthenticatorPassesRequestContextToRefresh(t *testing.T) {
	type ctxKey struct{}
	var got any
	s := &ctxSessionFake{onRefresh: func(ctx context.Context) { got = ctx.Value(ctxKey{}) }}
	a := NewAuthenticator(func(string) Session { return s }, NewCookie(CookieConfig{}))
	req := authRequest("sealed-1")
	req = req.WithContext(context.WithValue(req.Context(), ctxKey{}, "marker"))

	_, _ = a.Authenticate(httptest.NewRecorder(), req)

	if got != "marker" {
		t.Errorf("context value = %v, want %q", got, "marker")
	}
}

type ctxSessionFake struct {
	onRefresh func(ctx context.Context)
}

func (s *ctxSessionFake) Authenticate() (string, error) {
	return "", ErrUnauthenticated
}

func (s *ctxSessionFake) Refresh(ctx context.Context) (string, string, error) {
	s.onRefresh(ctx)
	return "sealed-2", "user_1", nil
}
