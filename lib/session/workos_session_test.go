package session

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/workos/workos-go/v10"
)

func newFactory(ws *workosServer) Factory {
	return NewWorkOSFactory(ws.client(), testCookiePassword)
}

func TestWorkOSSessionAuthenticate(t *testing.T) {
	key, other := keys(t)

	tests := []struct {
		name    string
		sealed  func(t *testing.T) string
		jwks    int
		want    string
		wantErr error
	}{
		{
			name:   "valid session",
			sealed: func(t *testing.T) string { return validSession(t, "user_1") },
			want:   "user_1",
		},
		{
			name:    "expired access token",
			sealed:  func(t *testing.T) string { return expiredSession(t, "user_1") },
			wantErr: ErrUnauthenticated,
		},
		{
			name:    "garbage cookie",
			sealed:  func(t *testing.T) string { return "not-a-sealed-session" },
			wantErr: ErrUnauthenticated,
		},
		{
			name: "sealed with another password",
			sealed: func(t *testing.T) string {
				sealed, err := workos.SealSession(&workos.SessionData{
					AccessToken: signToken(t, key, "session_1", 5*time.Minute),
					User:        &workos.User{ID: "user_1"},
				}, "another-password-another-password")
				if err != nil {
					t.Fatal(err)
				}
				return sealed
			},
			wantErr: ErrUnauthenticated,
		},
		{
			name: "token signed with another key",
			sealed: func(t *testing.T) string {
				return seal(t, &workos.SessionData{
					AccessToken: signToken(t, other, "session_1", 5*time.Minute),
					User:        &workos.User{ID: "user_1"},
				})
			},
			wantErr: ErrUnauthenticated,
		},
		{
			name: "missing user",
			sealed: func(t *testing.T) string {
				return seal(t, &workos.SessionData{
					AccessToken: signToken(t, key, "session_1", 5*time.Minute),
				})
			},
			wantErr: ErrUnauthenticated,
		},
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

			got, err := newFactory(ws)(tt.sealed(t)).Authenticate()

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Authenticate() error = %v, want %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("Authenticate() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestWorkOSSessionRefreshSucceeds(t *testing.T) {
	key, _ := keys(t)
	ws := newWorkOSServer(t)
	newAccess := signToken(t, key, "session_1", 5*time.Minute)
	ws.setAuthResponse(http.StatusOK, authResponse(newAccess, "refresh_2", "user_1"))
	factory := newFactory(ws)

	sealed, userID, err := factory(expiredSession(t, "user_1")).Refresh(context.Background())

	if err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}
	if userID != "user_1" {
		t.Errorf("userID = %q, want %q", userID, "user_1")
	}
	got, err := factory(sealed).Authenticate()
	if err != nil || got != "user_1" {
		t.Errorf("Authenticate() on refreshed session = (%q, %v), want (%q, nil)", got, err, "user_1")
	}
	requests := ws.requests()
	if len(requests) != 1 {
		t.Fatalf("auth requests = %d, want 1", len(requests))
	}
	if requests[0]["grant_type"] != "refresh_token" || requests[0]["refresh_token"] != "refresh_1" {
		t.Errorf("refresh request = %v, want grant_type refresh_token with refresh_1", requests[0])
	}
}

func TestWorkOSSessionRefreshFailures(t *testing.T) {
	tests := []struct {
		name    string
		sealed  func(t *testing.T) string
		status  int
		body    any
		wantErr error
	}{
		{
			name:    "revoked refresh token",
			sealed:  func(t *testing.T) string { return expiredSession(t, "user_1") },
			status:  http.StatusBadRequest,
			body:    map[string]string{"error": "invalid_grant", "error_description": "revoked"},
			wantErr: ErrUnauthenticated,
		},
		{
			name:    "workos server error",
			sealed:  func(t *testing.T) string { return expiredSession(t, "user_1") },
			status:  http.StatusInternalServerError,
			body:    map[string]string{"message": "boom"},
			wantErr: ErrAuthUnavailable,
		},
		{
			name:    "response without user",
			sealed:  func(t *testing.T) string { return expiredSession(t, "user_1") },
			status:  http.StatusOK,
			body:    map[string]any{"access_token": "a", "refresh_token": "r"},
			wantErr: ErrUnauthenticated,
		},
		{
			name:    "garbage cookie",
			sealed:  func(t *testing.T) string { return "not-a-sealed-session" },
			status:  http.StatusOK,
			wantErr: ErrUnauthenticated,
		},
		{
			name: "session without refresh token",
			sealed: func(t *testing.T) string {
				key, _ := keys(t)
				return seal(t, &workos.SessionData{
					AccessToken: signToken(t, key, "session_1", -time.Hour),
					User:        &workos.User{ID: "user_1"},
				})
			},
			status:  http.StatusOK,
			wantErr: ErrUnauthenticated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ws := newWorkOSServer(t)
			ws.setAuthResponse(tt.status, tt.body)

			sealed, userID, err := newFactory(ws)(tt.sealed(t)).Refresh(context.Background())

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Refresh() error = %v, want %v", err, tt.wantErr)
			}
			if sealed != "" || userID != "" {
				t.Errorf("Refresh() = (%q, %q), want empty values", sealed, userID)
			}
		})
	}
}

func TestWorkOSSessionRefreshWhenServerIsDown(t *testing.T) {
	ws := newWorkOSServer(t)
	factory := newFactory(ws)
	sealed := expiredSession(t, "user_1")
	ws.Close()

	_, _, err := factory(sealed).Refresh(context.Background())

	if !errors.Is(err, ErrAuthUnavailable) {
		t.Fatalf("Refresh() error = %v, want %v", err, ErrAuthUnavailable)
	}
}
