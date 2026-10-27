package session

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

const testRedirectURI = "http://localhost:8080/auth/callback"

func newTestLogin(ws *workosServer) *Login {
	return NewLogin(ws.client(), testRedirectURI, testCookiePassword)
}

func TestLoginAuthorizationURL(t *testing.T) {
	ws := newWorkOSServer(t)

	raw := newTestLogin(ws).AuthorizationURL("state-123")

	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("url.Parse() error = %v", err)
	}
	if !strings.HasPrefix(raw, ws.URL+"/user_management/authorize") {
		t.Errorf("URL = %q, want prefix %q", raw, ws.URL+"/user_management/authorize")
	}
	want := map[string]string{
		"response_type": "code",
		"client_id":     testClientID,
		"provider":      "authkit",
		"state":         "state-123",
		"redirect_uri":  testRedirectURI,
	}
	for key, value := range want {
		if got := u.Query().Get(key); got != value {
			t.Errorf("query %q = %q, want %q", key, got, value)
		}
	}
}

func TestLoginExchangeSucceeds(t *testing.T) {
	key, _ := keys(t)
	ws := newWorkOSServer(t)
	ws.setAuthResponse(http.StatusOK, authResponse(signToken(t, key, "session_1", 5*time.Minute), "refresh_1", "user_1"))

	result, err := newTestLogin(ws).Exchange(context.Background(), "code-123")

	if err != nil {
		t.Fatalf("Exchange() error = %v", err)
	}
	if result.UserID != "user_1" {
		t.Errorf("UserID = %q, want %q", result.UserID, "user_1")
	}
	if result.Email != "user_1@example.com" {
		t.Errorf("Email = %q, want %q", result.Email, "user_1@example.com")
	}
	if result.Sealed == "" {
		t.Fatal("Sealed is empty")
	}
	got, err := newFactory(ws)(result.Sealed).Authenticate()
	if err != nil || got.UserID != "user_1" || got.Email != "user_1@example.com" {
		t.Errorf("Authenticate() on sealed session = (%+v, %v), want user_1 with its email", got, err)
	}
	requests := ws.requests()
	if len(requests) != 1 {
		t.Fatalf("auth requests = %d, want 1", len(requests))
	}
	if requests[0]["grant_type"] != "authorization_code" || requests[0]["code"] != "code-123" {
		t.Errorf("request = %v, want authorization_code with code-123", requests[0])
	}
}

func TestLoginExchangeRejectsEmptyCodeWithoutCallingWorkOS(t *testing.T) {
	ws := newWorkOSServer(t)

	_, err := newTestLogin(ws).Exchange(context.Background(), "")

	if !errors.Is(err, ErrUnauthenticated) {
		t.Fatalf("Exchange() error = %v, want %v", err, ErrUnauthenticated)
	}
	if got := len(ws.requests()); got != 0 {
		t.Errorf("auth requests = %d, want 0", got)
	}
}

func TestLoginExchangeFailures(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		body    any
		wantErr error
	}{
		{
			name:    "invalid code",
			status:  http.StatusBadRequest,
			body:    map[string]string{"error": "invalid_grant"},
			wantErr: ErrUnauthenticated,
		},
		{
			name:    "unauthorized client",
			status:  http.StatusUnauthorized,
			body:    map[string]string{"error": "invalid_client"},
			wantErr: ErrUnauthenticated,
		},
		{
			name:    "workos server error",
			status:  http.StatusInternalServerError,
			body:    map[string]string{"message": "boom"},
			wantErr: ErrAuthUnavailable,
		},
		{
			name:    "response without user",
			status:  http.StatusOK,
			body:    map[string]any{"access_token": "a", "refresh_token": "r"},
			wantErr: ErrUnauthenticated,
		},
		{
			name:    "response without access token",
			status:  http.StatusOK,
			body:    map[string]any{"user": map[string]string{"id": "user_1"}, "refresh_token": "r"},
			wantErr: ErrUnauthenticated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ws := newWorkOSServer(t)
			ws.setAuthResponse(tt.status, tt.body)

			result, err := newTestLogin(ws).Exchange(context.Background(), "code-123")

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Exchange() error = %v, want %v", err, tt.wantErr)
			}
			if result != (Result{}) {
				t.Errorf("Exchange() result = %+v, want zero value", result)
			}
		})
	}
}

func TestLoginExchangeWhenServerIsDown(t *testing.T) {
	ws := newWorkOSServer(t)
	login := newTestLogin(ws)
	ws.Close()

	_, err := login.Exchange(context.Background(), "code-123")

	if !errors.Is(err, ErrAuthUnavailable) {
		t.Fatalf("Exchange() error = %v, want %v", err, ErrAuthUnavailable)
	}
}

func TestLoginLogoutURL(t *testing.T) {
	ws := newWorkOSServer(t)
	login := newTestLogin(ws)

	raw, err := login.LogoutURL(context.Background(), validSession(t, "user_1"), "http://localhost:3000/")

	if err != nil {
		t.Fatalf("LogoutURL() error = %v", err)
	}
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("url.Parse() error = %v", err)
	}
	if u.Path != "/user_management/sessions/logout" {
		t.Errorf("Path = %q, want %q", u.Path, "/user_management/sessions/logout")
	}
	if got := u.Query().Get("session_id"); got != "session_1" {
		t.Errorf("session_id = %q, want %q", got, "session_1")
	}
	if got := u.Query().Get("return_to"); got != "http://localhost:3000/" {
		t.Errorf("return_to = %q, want %q", got, "http://localhost:3000/")
	}
}

func TestLoginLogoutURLWithExpiredAccessToken(t *testing.T) {
	ws := newWorkOSServer(t)

	raw, err := newTestLogin(ws).LogoutURL(context.Background(), expiredSession(t, "user_1"), "")

	if err != nil {
		t.Fatalf("LogoutURL() error = %v", err)
	}
	if !strings.Contains(raw, "session_id=session_1") {
		t.Errorf("URL = %q, want session_id=session_1", raw)
	}
	if strings.Contains(raw, "return_to") {
		t.Errorf("URL = %q, want no return_to", raw)
	}
}

func TestLoginLogoutURLFailures(t *testing.T) {
	tests := []struct {
		name    string
		sealed  func(t *testing.T) string
		jwks    int
		wantErr error
	}{
		{name: "empty cookie", sealed: func(_ *testing.T) string { return "" }, wantErr: ErrUnauthenticated},
		{name: "garbage cookie", sealed: func(_ *testing.T) string { return "not-a-sealed-session" }, wantErr: ErrUnauthenticated},
		{
			name:    "signing keys unavailable",
			sealed:  func(t *testing.T) string { return validSession(t, "user_1") },
			jwks:    http.StatusInternalServerError,
			wantErr: ErrAuthUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ws := newWorkOSServer(t)
			if tt.jwks != 0 {
				ws.setJWKSStatus(tt.jwks)
			}

			got, err := newTestLogin(ws).LogoutURL(context.Background(), tt.sealed(t), "http://localhost:3000/")

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("LogoutURL() error = %v, want %v", err, tt.wantErr)
			}
			if got != "" {
				t.Errorf("LogoutURL() = %q, want empty", got)
			}
		})
	}
}
