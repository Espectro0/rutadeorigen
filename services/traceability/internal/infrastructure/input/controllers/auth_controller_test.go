package controllers_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"rutadeorigen/lib/session"
	"rutadeorigen/traceability/internal/infrastructure/input/controllers"
)

const (
	authSuccessURL = "https://app.example.com/home"
	authFailureURL = "https://app.example.com/login"
	authLogoutURL  = "https://app.example.com/"
)

type fakeLoginFlow struct {
	authURL        string
	exchangeResult session.Result
	exchangeErr    error
	logoutURL      string
	logoutErr      error

	authURLStates  []string
	exchangeCodes  []string
	logoutSealed   string
	logoutReturnTo string
	logoutCalls    int
}

func (f *fakeLoginFlow) AuthorizationURL(state string) string {
	f.authURLStates = append(f.authURLStates, state)
	return f.authURL + "?state=" + url.QueryEscape(state)
}

func (f *fakeLoginFlow) Exchange(_ context.Context, code string) (session.Result, error) {
	f.exchangeCodes = append(f.exchangeCodes, code)
	if f.exchangeErr != nil {
		return session.Result{}, f.exchangeErr
	}
	return f.exchangeResult, nil
}

func (f *fakeLoginFlow) LogoutURL(_ context.Context, sealed, returnTo string) (string, error) {
	f.logoutCalls++
	f.logoutSealed = sealed
	f.logoutReturnTo = returnTo
	if f.logoutErr != nil {
		return "", f.logoutErr
	}
	return f.logoutURL, nil
}

type fakeStateGuard struct {
	issueValue string
	issueErr   error
	verifyErr  error

	issueCalls     int
	verifyCalls    int
	verifyReceived string
}

func (f *fakeStateGuard) Issue(_ http.ResponseWriter) (string, error) {
	f.issueCalls++
	if f.issueErr != nil {
		return "", f.issueErr
	}
	return f.issueValue, nil
}

func (f *fakeStateGuard) Verify(_ http.ResponseWriter, _ *http.Request, received string) error {
	f.verifyCalls++
	f.verifyReceived = received
	return f.verifyErr
}

type fakeSessionCookie struct {
	readValue string
	readOK    bool
	setErr    error

	sets   []string
	clears int
}

func (f *fakeSessionCookie) Read(_ *http.Request) (string, bool) {
	return f.readValue, f.readOK
}

func (f *fakeSessionCookie) Set(_ http.ResponseWriter, sealed string) error {
	if f.setErr != nil {
		return f.setErr
	}
	f.sets = append(f.sets, sealed)
	return nil
}

func (f *fakeSessionCookie) Clear(_ http.ResponseWriter) {
	f.clears++
}

type authFixture struct {
	flow   *fakeLoginFlow
	state  *fakeStateGuard
	cookie *fakeSessionCookie
	mux    *http.ServeMux
}

func newAuthFixture(t *testing.T) *authFixture {
	t.Helper()

	f := &authFixture{
		flow: &fakeLoginFlow{
			authURL:        "https://auth.workos.example/authorize",
			exchangeResult: session.Result{Sealed: "sealed-1", UserID: "user_01", Email: "producer@example.com"},
			logoutURL:      "https://auth.workos.example/logout",
		},
		state:  &fakeStateGuard{issueValue: "state-1"},
		cookie: &fakeSessionCookie{},
	}

	controller, err := controllers.NewAuthController(f.flow, f.state, f.cookie, controllers.AuthRedirects{
		Success: authSuccessURL,
		Failure: authFailureURL,
		Logout:  authLogoutURL,
	}, discardLogger())
	if err != nil {
		t.Fatalf("NewAuthController: %v", err)
	}

	f.mux = http.NewServeMux()
	controller.Register(f.mux)
	return f
}

func (f *authFixture) do(method, target string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	f.mux.ServeHTTP(rec, httptest.NewRequest(method, target, nil))
	return rec
}

func (f *authFixture) get(target string) *httptest.ResponseRecorder {
	return f.do(http.MethodGet, target)
}

func locationOf(t *testing.T, rec *httptest.ResponseRecorder) *url.URL {
	t.Helper()

	raw := rec.Header().Get("Location")
	if raw == "" {
		t.Fatalf("no Location header, status %d", rec.Code)
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("invalid Location %q: %v", raw, err)
	}
	return parsed
}

func assertRedirect(t *testing.T, rec *httptest.ResponseRecorder, want string) {
	t.Helper()

	if rec.Code != http.StatusFound {
		t.Fatalf("status: got %d, want %d", rec.Code, http.StatusFound)
	}
	if got := rec.Header().Get("Location"); got != want {
		t.Errorf("Location: got %q, want %q", got, want)
	}
}

func assertFailureRedirect(t *testing.T, rec *httptest.ResponseRecorder, reason string) {
	t.Helper()

	if rec.Code != http.StatusFound {
		t.Fatalf("status: got %d, want %d", rec.Code, http.StatusFound)
	}
	location := locationOf(t, rec)
	if got := location.Scheme + "://" + location.Host + location.Path; got != authFailureURL {
		t.Errorf("failure target: got %q, want %q", got, authFailureURL)
	}
	if got := location.Query().Get("error"); got != reason {
		t.Errorf("error reason: got %q, want %q", got, reason)
	}
}

func TestNewAuthControllerValidatesRedirects(t *testing.T) {
	valid := controllers.AuthRedirects{Success: authSuccessURL, Failure: authFailureURL, Logout: authLogoutURL}
	invalid := []string{"", "/home", "app.example.com", "javascript:alert(1)", "ftp://app.example.com", "https://", "http://%zz"}

	t.Run("accepts absolute http and https urls", func(t *testing.T) {
		local := controllers.AuthRedirects{Success: "http://localhost:5173/home", Failure: "http://localhost:5173/login", Logout: "http://localhost:5173/"}
		for _, redirects := range []controllers.AuthRedirects{valid, local} {
			if _, err := controllers.NewAuthController(&fakeLoginFlow{}, &fakeStateGuard{}, &fakeSessionCookie{}, redirects, discardLogger()); err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		}
	})

	fields := map[string]func(controllers.AuthRedirects, string) controllers.AuthRedirects{
		"success": func(r controllers.AuthRedirects, v string) controllers.AuthRedirects { r.Success = v; return r },
		"failure": func(r controllers.AuthRedirects, v string) controllers.AuthRedirects { r.Failure = v; return r },
		"logout":  func(r controllers.AuthRedirects, v string) controllers.AuthRedirects { r.Logout = v; return r },
	}

	for field, set := range fields {
		for _, value := range invalid {
			t.Run(fmt.Sprintf("rejects %s %q", field, value), func(t *testing.T) {
				controller, err := controllers.NewAuthController(&fakeLoginFlow{}, &fakeStateGuard{}, &fakeSessionCookie{}, set(valid, value), discardLogger())

				if err == nil {
					t.Fatal("expected an error")
				}
				if controller != nil {
					t.Error("expected no controller on error")
				}
				if got := err.Error(); !strings.Contains(got, field) {
					t.Errorf("error %q must name the field %q", got, field)
				}
			})
		}
	}
}

func TestAuthLoginRedirectsToTheProvider(t *testing.T) {
	f := newAuthFixture(t)

	rec := f.get("/auth/login")

	assertRedirect(t, rec, "https://auth.workos.example/authorize?state=state-1")
	if len(f.flow.authURLStates) != 1 || f.flow.authURLStates[0] != "state-1" {
		t.Errorf("authorization url built with %v, want [state-1]", f.flow.authURLStates)
	}
	if got := rec.Header().Get("Cache-Control"); got != "no-store" {
		t.Errorf("Cache-Control: got %q, want no-store", got)
	}
}

func TestAuthLoginFailsWhenTheStateCannotBeIssued(t *testing.T) {
	f := newAuthFixture(t)
	f.state.issueErr = errors.New("entropy exhausted")

	rec := f.get("/auth/login")

	assertFailureRedirect(t, rec, "internal")
	if len(f.flow.authURLStates) != 0 {
		t.Error("the provider url must not be built without a state")
	}
}

func TestAuthCallbackSucceeds(t *testing.T) {
	f := newAuthFixture(t)

	rec := f.get("/auth/callback?code=code-123&state=state-1")

	assertRedirect(t, rec, authSuccessURL)
	if got := rec.Header().Get("Cache-Control"); got != "no-store" {
		t.Errorf("Cache-Control: got %q, want no-store", got)
	}
	if f.state.verifyCalls != 1 || f.state.verifyReceived != "state-1" {
		t.Errorf("state verified %d times with %q, want once with state-1", f.state.verifyCalls, f.state.verifyReceived)
	}
	if len(f.flow.exchangeCodes) != 1 || f.flow.exchangeCodes[0] != "code-123" {
		t.Errorf("exchanged codes %v, want [code-123]", f.flow.exchangeCodes)
	}
	if len(f.cookie.sets) != 1 || f.cookie.sets[0] != "sealed-1" {
		t.Errorf("session cookie sets %v, want [sealed-1]", f.cookie.sets)
	}
}

func TestAuthCallbackIgnoresRedirectParametersFromTheQuery(t *testing.T) {
	f := newAuthFixture(t)

	rec := f.get("/auth/callback?code=code-123&state=state-1&next=https://evil.example&redirect_uri=https://evil.example&return_to=https://evil.example")

	assertRedirect(t, rec, authSuccessURL)
}

func TestAuthCallbackRejectsAnInvalidState(t *testing.T) {
	f := newAuthFixture(t)
	f.state.verifyErr = session.ErrInvalidState

	rec := f.get("/auth/callback?code=code-123&state=forged")

	assertFailureRedirect(t, rec, "invalid_state")
	if len(f.flow.exchangeCodes) != 0 {
		t.Error("the code must not be exchanged with an invalid state")
	}
	if len(f.cookie.sets) != 0 {
		t.Error("no session cookie may be set with an invalid state")
	}
}

func TestAuthCallbackWhenTheProviderDeniesAccess(t *testing.T) {
	f := newAuthFixture(t)

	rec := f.get("/auth/callback?error=access_denied&error_description=nope&state=state-1")

	assertFailureRedirect(t, rec, "denied")
	if f.state.verifyCalls != 1 {
		t.Errorf("state verifications: got %d, want 1 so the state cookie is always cleared", f.state.verifyCalls)
	}
	if len(f.flow.exchangeCodes) != 0 || len(f.cookie.sets) != 0 {
		t.Error("nothing may be exchanged or stored when access is denied")
	}
}

func TestAuthCallbackMapsExchangeErrors(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		reason string
	}{
		{"unauthenticated", fmt.Errorf("%w: invalid code", session.ErrUnauthenticated), "unauthenticated"},
		{"unavailable", fmt.Errorf("%w: timeout", session.ErrAuthUnavailable), "unavailable"},
		{"unclassified", errors.New("boom"), "internal"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newAuthFixture(t)
			f.flow.exchangeErr = tt.err

			rec := f.get("/auth/callback?code=code-123&state=state-1")

			assertFailureRedirect(t, rec, tt.reason)
			if len(f.cookie.sets) != 0 {
				t.Error("no session cookie may be set when the exchange fails")
			}
		})
	}
}

func TestAuthCallbackFailsWhenTheSessionCookieCannotBeStored(t *testing.T) {
	f := newAuthFixture(t)
	f.cookie.setErr = session.ErrCookieTooLarge

	rec := f.get("/auth/callback?code=code-123&state=state-1")

	assertFailureRedirect(t, rec, "internal")
}

func TestAuthFailureRedirectKeepsTheConfiguredQuery(t *testing.T) {
	f := &authFixture{flow: &fakeLoginFlow{}, state: &fakeStateGuard{verifyErr: session.ErrInvalidState}, cookie: &fakeSessionCookie{}}
	controller, err := controllers.NewAuthController(f.flow, f.state, f.cookie, controllers.AuthRedirects{
		Success: authSuccessURL,
		Failure: authFailureURL + "?source=auth",
		Logout:  authLogoutURL,
	}, discardLogger())
	if err != nil {
		t.Fatal(err)
	}
	f.mux = http.NewServeMux()
	controller.Register(f.mux)

	rec := f.get("/auth/callback?code=c&state=s")

	query := locationOf(t, rec).Query()
	if query.Get("source") != "auth" || query.Get("error") != "invalid_state" {
		t.Errorf("query: got %v, want source=auth and error=invalid_state", query)
	}
}

func TestAuthLogoutWithSession(t *testing.T) {
	f := newAuthFixture(t)
	f.cookie.readValue, f.cookie.readOK = "sealed-1", true

	rec := f.get("/auth/logout")

	assertRedirect(t, rec, "https://auth.workos.example/logout")
	if f.cookie.clears != 1 {
		t.Errorf("cookie clears: got %d, want 1", f.cookie.clears)
	}
	if f.flow.logoutCalls != 1 || f.flow.logoutSealed != "sealed-1" || f.flow.logoutReturnTo != authLogoutURL {
		t.Errorf("logout url requested %d times with (%q, %q), want once with (sealed-1, %s)", f.flow.logoutCalls, f.flow.logoutSealed, f.flow.logoutReturnTo, authLogoutURL)
	}
	if got := rec.Header().Get("Cache-Control"); got != "no-store" {
		t.Errorf("Cache-Control: got %q, want no-store", got)
	}
}

func TestAuthLogoutWithoutSession(t *testing.T) {
	f := newAuthFixture(t)

	rec := f.get("/auth/logout")

	assertRedirect(t, rec, authLogoutURL)
	if f.flow.logoutCalls != 0 {
		t.Error("the provider must not be called without a session")
	}
	if f.cookie.clears != 1 {
		t.Errorf("cookie clears: got %d, want 1", f.cookie.clears)
	}
}

func TestAuthLogoutStillClearsTheCookieWhenTheProviderFails(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{"unauthenticated", fmt.Errorf("%w: bad session", session.ErrUnauthenticated)},
		{"unavailable", fmt.Errorf("%w: down", session.ErrAuthUnavailable)},
		{"unclassified", errors.New("boom")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newAuthFixture(t)
			f.cookie.readValue, f.cookie.readOK = "sealed-1", true
			f.flow.logoutErr = tt.err

			rec := f.get("/auth/logout")

			assertRedirect(t, rec, authLogoutURL)
			if f.cookie.clears != 1 {
				t.Errorf("cookie clears: got %d, want 1", f.cookie.clears)
			}
		})
	}
}

func TestAuthRoutesOnlyAcceptGet(t *testing.T) {
	for _, path := range []string{"/auth/login", "/auth/callback", "/auth/logout"} {
		for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete} {
			t.Run(method+" "+path, func(t *testing.T) {
				f := newAuthFixture(t)

				rec := f.do(method, path)

				if rec.Code != http.StatusMethodNotAllowed {
					t.Errorf("status: got %d, want %d", rec.Code, http.StatusMethodNotAllowed)
				}
				if f.state.issueCalls != 0 || f.state.verifyCalls != 0 || f.flow.logoutCalls != 0 || f.cookie.clears != 0 {
					t.Error("a rejected method must not run the flow")
				}
			})
		}
	}
}
