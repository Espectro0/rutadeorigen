package main

import (
	"io"
	"log/slog"
	"strings"
	"testing"
)

var authEnvKeys = []string{
	"AUTH_MODE",
	"WORKOS_API_KEY",
	"WORKOS_CLIENT_ID",
	"WORKOS_REDIRECT_URI",
	"WORKOS_COOKIE_PASSWORD",
	"AUTH_SUCCESS_URL",
	"AUTH_FAILURE_URL",
	"AUTH_LOGOUT_URL",
	"COOKIE_SECURE",
	"COOKIE_DOMAIN",
}

func validAuthEnv() map[string]string {
	return map[string]string{
		"AUTH_MODE":              "oidc",
		"WORKOS_API_KEY":         "sk_test_123",
		"WORKOS_CLIENT_ID":       "client_123",
		"WORKOS_REDIRECT_URI":    "http://localhost:8080/auth/callback",
		"WORKOS_COOKIE_PASSWORD": strings.Repeat("a", 32),
		"AUTH_SUCCESS_URL":       "http://localhost:5173/home",
		"AUTH_FAILURE_URL":       "http://localhost:5173/login",
		"AUTH_LOGOUT_URL":        "http://localhost:5173/",
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

func TestLoadAuthOIDCWithAValidConfiguration(t *testing.T) {
	setAuthEnv(t, nil)

	got, err := loadAuth(discardAuthLogger(), "postgres")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.mode != authModeOIDC {
		t.Errorf("mode: got %q, want %q", got.mode, authModeOIDC)
	}
	if got.authenticator == nil || got.routes == nil {
		t.Error("the oidc mode must provide an authenticator and the auth routes")
	}
}

func TestLoadAuthDefaultsToOIDC(t *testing.T) {
	clearAuthEnv(t)

	_, err := loadAuth(discardAuthLogger(), "memory")

	if err == nil || !strings.Contains(err.Error(), "WORKOS_API_KEY") {
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
	for _, key := range []string{
		"WORKOS_API_KEY",
		"WORKOS_CLIENT_ID",
		"WORKOS_REDIRECT_URI",
		"WORKOS_COOKIE_PASSWORD",
		"AUTH_SUCCESS_URL",
		"AUTH_FAILURE_URL",
		"AUTH_LOGOUT_URL",
	} {
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
		{"relative success url", map[string]string{"AUTH_SUCCESS_URL": "/home"}, "success"},
		{"relative failure url", map[string]string{"AUTH_FAILURE_URL": "/login"}, "failure"},
		{"relative logout url", map[string]string{"AUTH_LOGOUT_URL": "/"}, "logout"},
		{"unsafe success url", map[string]string{"AUTH_SUCCESS_URL": "javascript:alert(1)"}, "success"},
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
		if got.authenticator != nil || got.routes != nil {
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

func TestDevHeaderModeHasNoSessionMiddleware(t *testing.T) {
	requireActor, authenticated := authn{mode: authModeDevHeader}.middlewares(nil, nil)

	if requireActor == nil {
		t.Error("the dev-header mode must still provide a RequireActor middleware")
	}
	if authenticated != nil {
		t.Error("the dev-header mode must not provide a session middleware")
	}
}
