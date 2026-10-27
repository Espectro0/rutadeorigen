package main

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"rutadeorigen/lib/requestcontext"
	"rutadeorigen/lib/session"
	"rutadeorigen/traceability/internal/infrastructure/input/controllers"
	"rutadeorigen/traceability/internal/platform/catalog"
	"rutadeorigen/traceability/internal/platform/exception"
)

const (
	oidcSuccessURL    = "http://localhost:5173/home"
	oidcFailureURL    = "http://localhost:5173/login"
	oidcLogoutURL     = "http://localhost:5173/"
	oidcProviderURL   = "https://auth.example.com/authorize"
	sessionCookieName = "ro_session"
	stateCookieName   = "ro_oauth_state"
	sealedUnavailable = "sealed-unavailable"
	registerBody      = `{"type":"producer","name":"Finca Nueva SAS"}`
)

var oidcIdentities = map[string]requestcontext.Identity{
	"sealed-new":   {UserID: "user_new", Email: "  New.Producer@Example.com "},
	"sealed-other": {UserID: "user_other", Email: "other@example.com"},
	"sealed-seed":  {UserID: "seed-producer", Email: "seed@rutadeorigen.com", FirstName: namePtr("Juan"), LastName: namePtr("Ramírez")},
}

func namePtr(s string) *string { return &s }

type oidcFlow struct{}

func (oidcFlow) AuthorizationURL(state string) string {
	return oidcProviderURL + "?state=" + url.QueryEscape(state)
}

func (oidcFlow) Exchange(_ context.Context, code string) (session.Result, error) {
	sealed := "sealed-" + code
	identity, ok := oidcIdentities[sealed]
	if !ok {
		return session.Result{}, session.ErrUnauthenticated
	}
	return session.Result{Sealed: sealed, UserID: identity.UserID, Email: identity.Email}, nil
}

func (oidcFlow) LogoutURL(_ context.Context, _, returnTo string) (string, error) {
	return "https://auth.example.com/logout?return_to=" + url.QueryEscape(returnTo), nil
}

type oidcSession struct {
	sealed string
}

func (s oidcSession) Authenticate() (requestcontext.Identity, error) {
	if s.sealed == sealedUnavailable {
		return requestcontext.Identity{}, session.ErrAuthUnavailable
	}
	if identity, ok := oidcIdentities[s.sealed]; ok {
		return identity, nil
	}
	return requestcontext.Identity{}, session.ErrUnauthenticated
}

func (oidcSession) Refresh(context.Context) (string, requestcontext.Identity, error) {
	return "", requestcontext.Identity{}, session.ErrUnauthenticated
}

func newOIDCServer(t *testing.T) *http.ServeMux {
	t.Helper()

	messages, err := catalog.New()
	if err != nil {
		t.Fatalf("catalog.New: %v", err)
	}

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	cookie := session.NewCookie(session.CookieConfig{Secure: true})
	state := session.NewState(session.StateConfig{Secure: true})

	routes, err := controllers.NewAuthController(oidcFlow{}, state, cookie, controllers.AuthRedirects{
		Success: oidcSuccessURL,
		Failure: oidcFailureURL,
		Logout:  oidcLogoutURL,
	}, log)
	if err != nil {
		t.Fatalf("NewAuthController: %v", err)
	}

	authenticator := session.NewAuthenticator(func(sealed string) session.Session {
		return oidcSession{sealed: sealed}
	}, cookie)

	mux := http.NewServeMux()
	registerRoutes(mux, log, nil, messages, authn{mode: authModeOIDC, authenticator: authenticator, routes: routes})
	return mux
}

func oidcRequest(mux http.Handler, method, target, body string, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, target, reader)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	for _, c := range cookies {
		req.AddCookie(c)
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func sessionOf(sealed string) *http.Cookie {
	return &http.Cookie{Name: sessionCookieName, Value: sealed}
}

func cookieNamed(rec *httptest.ResponseRecorder, name string) *http.Cookie {
	for _, c := range rec.Result().Cookies() {
		if c.Name == name {
			return c
		}
	}
	return nil
}

type actorView struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

func decodeActor(t *testing.T, rec *httptest.ResponseRecorder) actorView {
	t.Helper()

	var a actorView
	if err := json.NewDecoder(rec.Body).Decode(&a); err != nil {
		t.Fatalf("body is not JSON: %v", err)
	}
	return a
}

func mustRegister(t *testing.T, mux http.Handler, sealed string) actorView {
	t.Helper()

	rec := oidcRequest(mux, http.MethodPost, "/me", registerBody, sessionOf(sealed))
	if rec.Code != http.StatusCreated {
		t.Fatalf("register %s: status %d (%s)", sealed, rec.Code, rec.Body.String())
	}
	return decodeActor(t, rec)
}

func TestOIDCFullLoginFlowEndToEnd(t *testing.T) {
	mux := newOIDCServer(t)

	login := oidcRequest(mux, http.MethodGet, "/auth/login", "")
	if login.Code != http.StatusFound {
		t.Fatalf("login status: got %d, want %d", login.Code, http.StatusFound)
	}
	location, err := url.Parse(login.Header().Get("Location"))
	if err != nil {
		t.Fatalf("invalid Location: %v", err)
	}
	if !strings.HasPrefix(location.String(), oidcProviderURL) {
		t.Errorf("login must redirect to the provider, got %q", location)
	}
	state := location.Query().Get("state")
	stateCookie := cookieNamed(login, stateCookieName)
	if state == "" || stateCookie == nil || stateCookie.Value != state {
		t.Fatalf("the state in the url and in the cookie must match, got %q / %+v", state, stateCookie)
	}
	if !stateCookie.HttpOnly {
		t.Error("the state cookie must be HttpOnly")
	}

	callback := oidcRequest(mux, http.MethodGet, "/auth/callback?code=new&state="+url.QueryEscape(state), "", stateCookie)
	if callback.Code != http.StatusFound || callback.Header().Get("Location") != oidcSuccessURL {
		t.Fatalf("callback: got %d to %q, want %d to %q", callback.Code, callback.Header().Get("Location"), http.StatusFound, oidcSuccessURL)
	}
	sessionCookie := cookieNamed(callback, sessionCookieName)
	if sessionCookie == nil || sessionCookie.Value != "sealed-new" {
		t.Fatalf("the callback must set the session cookie, got %+v", sessionCookie)
	}
	if !sessionCookie.HttpOnly || !sessionCookie.Secure || sessionCookie.SameSite != http.SameSiteLaxMode {
		t.Errorf("session cookie must be HttpOnly, Secure and Lax, got %+v", sessionCookie)
	}
	if cleared := cookieNamed(callback, stateCookieName); cleared == nil || cleared.MaxAge >= 0 {
		t.Errorf("the state cookie must be cleared after the callback, got %+v", cleared)
	}

	notRegistered := oidcRequest(mux, http.MethodGet, "/me", "", sessionCookie)
	assertErrorCode(t, notRegistered, http.StatusNotFound, string(exception.CodeActorNotFound))

	created := oidcRequest(mux, http.MethodPost, "/me", registerBody, sessionCookie)
	if created.Code != http.StatusCreated {
		t.Fatalf("register: status %d (%s)", created.Code, created.Body.String())
	}
	if created.Header().Get("Location") != "/me" {
		t.Errorf("Location: got %q, want /me", created.Header().Get("Location"))
	}
	registered := decodeActor(t, created)
	if registered.Type != "producer" || registered.Name != "Finca Nueva SAS" || registered.Status != "active" {
		t.Errorf("unexpected actor: %+v", registered)
	}
	if registered.Email != "new.producer@example.com" {
		t.Errorf("the email must come from the session and be normalized, got %q", registered.Email)
	}

	again := oidcRequest(mux, http.MethodPost, "/me", registerBody, sessionCookie)
	if again.Code != http.StatusOK {
		t.Fatalf("second register: status %d, want %d", again.Code, http.StatusOK)
	}
	if got := decodeActor(t, again); got.ID != registered.ID {
		t.Errorf("registering twice must return the same actor, got %q and %q", got.ID, registered.ID)
	}

	me := oidcRequest(mux, http.MethodGet, "/me", "", sessionCookie)
	if me.Code != http.StatusOK {
		t.Fatalf("me: status %d (%s)", me.Code, me.Body.String())
	}
	if got := decodeActor(t, me); got != registered {
		t.Errorf("me: got %+v, want %+v", got, registered)
	}

	farm := oidcRequest(mux, http.MethodPost, "/farms", farmBody("La Esperanza", 1650), sessionCookie)
	if farm.Code != http.StatusCreated {
		t.Fatalf("create farm: status %d (%s)", farm.Code, farm.Body.String())
	}
	var createdFarm farmView
	if err := json.NewDecoder(farm.Body).Decode(&createdFarm); err != nil {
		t.Fatalf("body is not JSON: %v", err)
	}
	if createdFarm.ProducerID != registered.ID {
		t.Errorf("the farm must belong to the actor of the session, got %q want %q", createdFarm.ProducerID, registered.ID)
	}

	farms := decodeFarms(t, oidcRequest(mux, http.MethodGet, "/farms", "", sessionCookie))
	if len(farms.Items) != 1 || farms.Items[0].ID != createdFarm.ID {
		t.Errorf("the producer must see only its own farm, got %+v", farms.Items)
	}

	logout := oidcRequest(mux, http.MethodGet, "/auth/logout", "", sessionCookie)
	if logout.Code != http.StatusFound || !strings.HasPrefix(logout.Header().Get("Location"), "https://auth.example.com/logout") {
		t.Errorf("logout: got %d to %q", logout.Code, logout.Header().Get("Location"))
	}
	if cleared := cookieNamed(logout, sessionCookieName); cleared == nil || cleared.MaxAge >= 0 || cleared.Value != "" {
		t.Errorf("logout must clear the session cookie, got %+v", cleared)
	}
}

func TestOIDCRejectsRequestsWithoutAValidSession(t *testing.T) {
	tests := []struct {
		name   string
		method string
		target string
		body   string
	}{
		{"create farm", http.MethodPost, "/farms", farmBody("La Esperanza", 1650)},
		{"list farms", http.MethodGet, "/farms", ""},
		{"create lot", http.MethodPost, "/lots", lotBody(seedFarmID.String(), "LOT-001", "washed", "2026-01-15")},
		{"get me", http.MethodGet, "/me", ""},
		{"register", http.MethodPost, "/me", registerBody},
	}
	cookies := map[string][]*http.Cookie{
		"no cookie":      nil,
		"unknown cookie": {sessionOf("sealed-garbage")},
	}

	for _, tt := range tests {
		for cookieName, cookie := range cookies {
			t.Run(tt.name+" with "+cookieName, func(t *testing.T) {
				rec := oidcRequest(newOIDCServer(t), tt.method, tt.target, tt.body, cookie...)

				assertErrorCode(t, rec, http.StatusUnauthorized, string(exception.CodeUnauthorized))
			})
		}
	}
}

func TestOIDCIgnoresTheActorHeader(t *testing.T) {
	mux := newOIDCServer(t)

	req := httptest.NewRequest(http.MethodPost, "/farms", strings.NewReader(farmBody("La Esperanza", 1650)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Actor-ID", producerHeader)
	req.AddCookie(sessionOf("sealed-new"))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	assertErrorCode(t, rec, http.StatusNotFound, string(exception.CodeActorNotFound))
}

func TestOIDCActorHeaderAloneDoesNotAuthenticate(t *testing.T) {
	mux := newOIDCServer(t)

	req := httptest.NewRequest(http.MethodPost, "/farms", strings.NewReader(farmBody("La Esperanza", 1650)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Actor-ID", producerHeader)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	assertErrorCode(t, rec, http.StatusUnauthorized, string(exception.CodeUnauthorized))
}

func TestOIDCUserWithAnExistingActor(t *testing.T) {
	mux := newOIDCServer(t)

	me := oidcRequest(mux, http.MethodGet, "/me", "", sessionOf("sealed-seed"))
	if me.Code != http.StatusOK {
		t.Fatalf("me: status %d (%s)", me.Code, me.Body.String())
	}
	got := decodeActor(t, me)
	if got.ID != producerHeader || got.Type != "producer" {
		t.Errorf("the seed user must resolve to the seed producer, got %+v", got)
	}
	if got.FirstName != "Juan" || got.LastName != "Ramírez" {
		t.Errorf("me must return the names of the session, got (%q, %q)", got.FirstName, got.LastName)
	}

	farm := oidcRequest(mux, http.MethodPost, "/farms", farmBody("La Esperanza", 1650), sessionOf("sealed-seed"))
	if farm.Code != http.StatusCreated {
		t.Fatalf("create farm: status %d (%s)", farm.Code, farm.Body.String())
	}

	again := oidcRequest(mux, http.MethodPost, "/me", registerBody, sessionOf("sealed-seed"))
	if again.Code != http.StatusOK {
		t.Fatalf("register of an existing actor: status %d, want %d", again.Code, http.StatusOK)
	}
	if got := decodeActor(t, again); got.ID != producerHeader || got.Name != "Productor de prueba" {
		t.Errorf("registering must not change the stored actor, got %+v", got)
	}
}

func TestOIDCUsersAreIsolated(t *testing.T) {
	mux := newOIDCServer(t)
	first := mustRegister(t, mux, "sealed-new")
	second := mustRegister(t, mux, "sealed-other")

	if first.ID == second.ID {
		t.Fatal("different users must get different actors")
	}
	if second.Email != "other@example.com" {
		t.Errorf("each actor must keep the email of its own session, got %q", second.Email)
	}

	farm := oidcRequest(mux, http.MethodPost, "/farms", farmBody("La Esperanza", 1650), sessionOf("sealed-new"))
	if farm.Code != http.StatusCreated {
		t.Fatalf("create farm: status %d (%s)", farm.Code, farm.Body.String())
	}

	others := decodeFarms(t, oidcRequest(mux, http.MethodGet, "/farms", "", sessionOf("sealed-other")))
	if len(others.Items) != 0 {
		t.Errorf("a user must not see the farms of another, got %+v", others.Items)
	}
}

func TestOIDCRegisterRejectsInvalidData(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		want     int
		wantCode string
	}{
		{"unknown type", `{"type":"farmer","name":"Finca Nueva SAS"}`, http.StatusBadRequest, string(exception.CodeActorInvalidType)},
		{"empty type", `{"type":"","name":"Finca Nueva SAS"}`, http.StatusBadRequest, string(exception.CodeActorInvalidType)},
		{"short name", `{"type":"producer","name":"ab"}`, http.StatusBadRequest, string(exception.CodeActorInvalidName)},
		{"email in the body", `{"type":"producer","name":"Finca Nueva SAS","email":"x@example.com"}`, http.StatusBadRequest, string(exception.CodeInvalidRequestBody)},
		{"user id in the body", `{"type":"producer","name":"Finca Nueva SAS","user_id":"user_other"}`, http.StatusBadRequest, string(exception.CodeInvalidRequestBody)},
		{"malformed json", `{"type":`, http.StatusBadRequest, string(exception.CodeInvalidRequestBody)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := newOIDCServer(t)

			rec := oidcRequest(mux, http.MethodPost, "/me", tt.body, sessionOf("sealed-new"))

			assertErrorCode(t, rec, tt.want, tt.wantCode)
			if me := oidcRequest(mux, http.MethodGet, "/me", "", sessionOf("sealed-new")); me.Code != http.StatusNotFound {
				t.Errorf("a rejected registration must not create the actor, /me answered %d", me.Code)
			}
		})
	}
}

func TestOIDCReportsWhenTheProviderIsUnavailable(t *testing.T) {
	tests := []struct {
		name   string
		method string
		target string
		body   string
	}{
		{"get me", http.MethodGet, "/me", ""},
		{"register", http.MethodPost, "/me", registerBody},
		{"create farm", http.MethodPost, "/farms", farmBody("La Esperanza", 1650)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := oidcRequest(newOIDCServer(t), tt.method, tt.target, tt.body, sessionOf(sealedUnavailable))

			assertErrorCode(t, rec, http.StatusServiceUnavailable, string(exception.CodeAuthUnavailable))
			if cleared := cookieNamed(rec, sessionCookieName); cleared != nil {
				t.Errorf("an unavailable provider must not clear the session, got %+v", cleared)
			}
		})
	}
}

func TestOIDCCallbackRejectsAnInvalidState(t *testing.T) {
	mux := newOIDCServer(t)

	tests := []struct {
		name    string
		target  string
		cookies []*http.Cookie
		reason  string
	}{
		{"no state cookie", "/auth/callback?code=new&state=forged", nil, "invalid_state"},
		{"different state", "/auth/callback?code=new&state=forged", []*http.Cookie{{Name: stateCookieName, Value: "other"}}, "invalid_state"},
		{"no state in the url", "/auth/callback?code=new", []*http.Cookie{{Name: stateCookieName, Value: "state-1"}}, "invalid_state"},
		{"unknown code", "/auth/callback?code=unknown&state=state-1", []*http.Cookie{{Name: stateCookieName, Value: "state-1"}}, "unauthenticated"},
		{"provider error", "/auth/callback?error=access_denied&state=state-1", []*http.Cookie{{Name: stateCookieName, Value: "state-1"}}, "denied"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := oidcRequest(mux, http.MethodGet, tt.target, "", tt.cookies...)

			if rec.Code != http.StatusFound {
				t.Fatalf("status: got %d, want %d", rec.Code, http.StatusFound)
			}
			location, err := url.Parse(rec.Header().Get("Location"))
			if err != nil {
				t.Fatalf("invalid Location: %v", err)
			}
			if location.Host != "localhost:5173" || location.Path != "/login" || location.Query().Get("error") != tt.reason {
				t.Errorf("got redirect %q, want the failure page with error=%s", location, tt.reason)
			}
			if cookieNamed(rec, sessionCookieName) != nil {
				t.Error("a failed login must not set the session cookie")
			}
		})
	}
}

func TestDevHeaderModeDoesNotExposeTheAuthRoutes(t *testing.T) {
	mux, _ := newTestServer(t)

	for _, target := range []string{"/me", "/auth/login", "/auth/callback", "/auth/logout"} {
		t.Run(target, func(t *testing.T) {
			rec := oidcRequest(mux, http.MethodGet, target, "")

			if rec.Code != http.StatusNotFound {
				t.Errorf("status: got %d, want %d", rec.Code, http.StatusNotFound)
			}
		})
	}
}

func TestOIDCMeReturnsEmptyNamesWhenTheSessionHasNone(t *testing.T) {
	mux := newOIDCServer(t)
	mustRegister(t, mux, "sealed-new")

	me := oidcRequest(mux, http.MethodGet, "/me", "", sessionOf("sealed-new"))

	if me.Code != http.StatusOK {
		t.Fatalf("me: status %d (%s)", me.Code, me.Body.String())
	}
	if got := decodeActor(t, me); got.FirstName != "" || got.LastName != "" {
		t.Errorf("got names (%q, %q), want empty", got.FirstName, got.LastName)
	}
}

func TestOIDCRegisterDoesNotReturnTheNames(t *testing.T) {
	mux := newOIDCServer(t)

	rec := oidcRequest(mux, http.MethodPost, "/me", registerBody, sessionOf("sealed-seed"))

	if rec.Code != http.StatusOK {
		t.Fatalf("register: status %d (%s)", rec.Code, rec.Body.String())
	}
	if got := decodeActor(t, rec); got.FirstName != "" || got.LastName != "" {
		t.Errorf("register must not return names, got (%q, %q)", got.FirstName, got.LastName)
	}
}
