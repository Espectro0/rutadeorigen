package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"rutadeorigen/lib/requestcontext"
	"rutadeorigen/lib/session"
	"rutadeorigen/qr/internal/application/input/dto/request"
	"rutadeorigen/qr/internal/application/input/dto/response"
	"rutadeorigen/qr/internal/infrastructure/input/controllers"
	"rutadeorigen/qr/internal/platform/catalog"
	"rutadeorigen/qr/internal/platform/exception"
)

var authEnvKeys = []string{
	"AUTH_MODE",
	"WORKOS_ISSUER",
	"WORKOS_API_KEY",
	"WORKOS_CLIENT_ID",
	"WORKOS_COOKIE_PASSWORD",
	"COOKIE_SECURE",
	"COOKIE_DOMAIN",
}

func validAuthEnv() map[string]string {
	return map[string]string{
		"AUTH_MODE":              "oidc",
		"WORKOS_ISSUER":          "https://api.workos.com/user_management/client_123",
		"WORKOS_API_KEY":         "sk_test_123",
		"WORKOS_CLIENT_ID":       "client_123",
		"WORKOS_COOKIE_PASSWORD": strings.Repeat("a", 32),
		"COOKIE_SECURE":          "false",
		"COOKIE_DOMAIN":          "",
	}
}

func setAuthEnv(t *testing.T, overrides map[string]string) {
	t.Helper()

	values := validAuthEnv()
	for key, value := range overrides {
		values[key] = value
	}
	for _, key := range authEnvKeys {
		t.Setenv(key, values[key])
	}
}

func clearAuthEnv(t *testing.T) {
	t.Helper()

	for _, key := range authEnvKeys {
		t.Setenv(key, "")
	}
}

func discardAuthLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func newAuthErrors(t *testing.T) *controllers.ErrorResponder {
	t.Helper()

	messages, err := catalog.New()
	if err != nil {
		t.Fatalf("catalog.New: %v", err)
	}
	return controllers.NewErrorResponder(discardAuthLogger(), messages)
}

type authenticatorFake struct {
	identity requestcontext.Identity
	err      error
	calls    int
}

func (a *authenticatorFake) Authenticate(_ http.ResponseWriter, _ *http.Request) (requestcontext.Identity, error) {
	a.calls++
	return a.identity, a.err
}

func TestLoadAuthOIDCWithAValidConfiguration(t *testing.T) {
	setAuthEnv(t, nil)

	got, err := loadAuth(discardAuthLogger(), "postgres")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.mode != authModeOIDC {
		t.Errorf("mode: got %q, want %q", got.mode, authModeOIDC)
	}
	if got.authenticator == nil {
		t.Error("the oidc mode must provide an authenticator")
	}
}

func TestLoadAuthOIDCDoesNotNeedTheApiKeyOrTheRoutes(t *testing.T) {
	setAuthEnv(t, map[string]string{"WORKOS_API_KEY": ""})

	if _, err := loadAuth(discardAuthLogger(), "postgres"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadAuthDefaultsToOIDC(t *testing.T) {
	clearAuthEnv(t)

	_, err := loadAuth(discardAuthLogger(), "memory")

	if err == nil || !strings.Contains(err.Error(), "WORKOS_CLIENT_ID") {
		t.Fatalf("without AUTH_MODE the oidc configuration must be required, got %v", err)
	}
}

func TestLoadAuthOIDCListsEveryMissingVariable(t *testing.T) {
	clearAuthEnv(t)
	t.Setenv("AUTH_MODE", "oidc")

	_, err := loadAuth(discardAuthLogger(), "postgres")

	if err == nil {
		t.Fatal("expected an error")
	}
	for _, key := range []string{"WORKOS_CLIENT_ID", "WORKOS_COOKIE_PASSWORD"} {
		if !strings.Contains(err.Error(), key) {
			t.Errorf("the error must mention %s, got %q", key, err)
		}
	}
}

func TestLoadAuthOIDCRejectsInvalidValues(t *testing.T) {
	tests := []struct {
		name      string
		overrides map[string]string
		want      string
	}{
		{"short cookie password", map[string]string{"WORKOS_COOKIE_PASSWORD": strings.Repeat("a", 31)}, "WORKOS_COOKIE_PASSWORD"},
		{"invalid COOKIE_SECURE", map[string]string{"COOKIE_SECURE": "maybe"}, "COOKIE_SECURE"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setAuthEnv(t, tt.overrides)

			_, err := loadAuth(discardAuthLogger(), "postgres")

			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("got error %v, want one mentioning %q", err, tt.want)
			}
		})
	}
}

func TestLoadAuthOIDCAcceptsSecureCookiesByDefault(t *testing.T) {
	setAuthEnv(t, map[string]string{"COOKIE_SECURE": ""})

	if _, err := loadAuth(discardAuthLogger(), "postgres"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadAuthDevHeader(t *testing.T) {
	clearAuthEnv(t)
	t.Setenv("AUTH_MODE", "dev-header")

	t.Run("is allowed with memory storage", func(t *testing.T) {
		got, err := loadAuth(discardAuthLogger(), "memory")

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.mode != authModeDevHeader {
			t.Errorf("mode: got %q, want %q", got.mode, authModeDevHeader)
		}
		if got.authenticator != nil {
			t.Error("the dev-header mode must not provide session authentication")
		}
	})

	t.Run("is refused with any other storage", func(t *testing.T) {
		for _, storage := range []string{"postgres", "", "other"} {
			if _, err := loadAuth(discardAuthLogger(), storage); err == nil || !strings.Contains(err.Error(), "STORAGE=memory") {
				t.Errorf("storage %q: got error %v, want one requiring STORAGE=memory", storage, err)
			}
		}
	})
}

func TestLoadAuthRejectsAnUnknownMode(t *testing.T) {
	clearAuthEnv(t)
	t.Setenv("AUTH_MODE", "none")

	_, err := loadAuth(discardAuthLogger(), "memory")

	if err == nil || !strings.Contains(err.Error(), "unknown AUTH_MODE") {
		t.Fatalf("got error %v, want an unknown AUTH_MODE error", err)
	}
}

func TestRequireActorInDevHeaderModeTrustsTheHeader(t *testing.T) {
	actorID := uuid.New()
	var seen uuid.UUID
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = requestcontext.ActorID(r.Context())
		w.WriteHeader(http.StatusNoContent)
	})
	handler := authn{mode: authModeDevHeader}.requireActor(newAuthErrors(t))(next)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(controllers.ActorIDHeader, actorID.String())
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent || seen != actorID {
		t.Errorf("got status %d and actor %s, want 204 and %s", rec.Code, seen, actorID)
	}
}

func TestRequireActorInDevHeaderModeRejectsRequestsWithoutTheHeader(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true })
	handler := authn{mode: authModeDevHeader}.requireActor(newAuthErrors(t))(next)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if called || rec.Code != http.StatusUnauthorized {
		t.Errorf("got status %d and called %v, want 401 and false", rec.Code, called)
	}
}

type resolverFake struct {
	byUser map[string]response.ResolvedActor
}

func (f resolverFake) Execute(_ context.Context, in request.ResolveActor) (response.ResolvedActor, error) {
	if actor, ok := f.byUser[in.UserID]; ok {
		return actor, nil
	}
	return response.ResolvedActor{}, exception.New(exception.CodeActorViewNotFound)
}

func TestRequireActorInOIDCModeTakesTheActorFromTheProjection(t *testing.T) {
	fake := &authenticatorFake{identity: requestcontext.Identity{UserID: "user_01", Email: "producer@example.com"}}
	projected := uuid.New()
	resolver := resolverFake{byUser: map[string]response.ResolvedActor{"user_01": {ActorID: projected, Role: "producer"}}}
	var seenActor uuid.UUID
	var seenUser, seenRole string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenActor = requestcontext.ActorID(r.Context())
		seenUser = requestcontext.UserID(r.Context())
		seenRole = requestcontext.ActorRole(r.Context())
		w.WriteHeader(http.StatusNoContent)
	})
	handler := authn{mode: authModeOIDC, authenticator: fake, resolver: resolver}.requireActor(newAuthErrors(t))(next)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status: got %d, want %d", rec.Code, http.StatusNoContent)
	}
	if seenActor != projected || seenRole != "producer" {
		t.Errorf("got actor %s role %q, want %s producer", seenActor, seenRole, projected)
	}
	if seenUser != "user_01" {
		t.Errorf("user in context: got %q, want user_01", seenUser)
	}
}

func TestRequireActorInOIDCModeRejectsAnUnknownActor(t *testing.T) {
	fake := &authenticatorFake{identity: requestcontext.Identity{UserID: "user_99"}}
	called := false
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true })
	handler := authn{mode: authModeOIDC, authenticator: fake, resolver: resolverFake{}}.requireActor(newAuthErrors(t))(next)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if called || rec.Code != http.StatusForbidden {
		t.Errorf("got status %d and called %v, want 403 and false", rec.Code, called)
	}
}

func TestRequireActorInOIDCModeIgnoresTheActorHeader(t *testing.T) {
	fake := &authenticatorFake{err: session.ErrUnauthenticated}
	called := false
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true })
	handler := authn{mode: authModeOIDC, authenticator: fake}.requireActor(newAuthErrors(t))(next)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(controllers.ActorIDHeader, uuid.NewString())
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if called || rec.Code != http.StatusUnauthorized {
		t.Errorf("got status %d and called %v, want 401 and false", rec.Code, called)
	}
}

func TestRequireActorInOIDCModeMapsAuthenticationErrors(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		want     int
		wantCode string
	}{
		{"unauthenticated", session.ErrUnauthenticated, http.StatusUnauthorized, "UNAUTHORIZED"},
		{"unavailable", session.ErrAuthUnavailable, http.StatusServiceUnavailable, "AUTH_UNAVAILABLE"},
		{"unknown error", errors.New("boom"), http.StatusInternalServerError, "INTERNAL"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &authenticatorFake{err: tt.err}
			called := false
			next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true })
			handler := authn{mode: authModeOIDC, authenticator: fake}.requireActor(newAuthErrors(t))(next)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

			if called {
				t.Error("the next handler must not run when authentication fails")
			}
			if rec.Code != tt.want {
				t.Errorf("status: got %d, want %d", rec.Code, tt.want)
			}
			var body map[string]string
			if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
				t.Fatalf("body is not JSON: %v", err)
			}
			if body["code"] != tt.wantCode {
				t.Errorf("code: got %q, want %q", body["code"], tt.wantCode)
			}
			if body["error"] == "" {
				t.Error("error message must not be empty")
			}
		})
	}
}
