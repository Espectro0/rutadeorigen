package controllers_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"rutadeorigen/lib/requestcontext"
	"rutadeorigen/lib/session"
	"rutadeorigen/qr/internal/infrastructure/input/controllers"
	"rutadeorigen/qr/internal/platform/exception"
)

func sessionRequest(userID string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/public/lots/abc/qr", nil)
	if userID == "" {
		return req
	}
	return req.WithContext(requestcontext.WithUserID(req.Context(), userID))
}

func serveHandler(handler http.Handler, req *http.Request) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func actorSeenBy(t *testing.T, userID string) uuid.UUID {
	t.Helper()

	var seen uuid.UUID
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = requestcontext.ActorID(r.Context())
		w.WriteHeader(http.StatusNoContent)
	})

	rec := serveHandler(controllers.ActorFromSession(newErrs(), next), sessionRequest(userID))

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status: got %d, want %d", rec.Code, http.StatusNoContent)
	}
	return seen
}

func TestActorFromSessionDerivesTheActorFromTheUser(t *testing.T) {
	got := actorSeenBy(t, "user_01")

	want := uuid.NewSHA1(uuid.NameSpaceURL, []byte("user_01"))
	if got != want {
		t.Errorf("actor in context: got %s, want %s", got, want)
	}
	if got == uuid.Nil {
		t.Error("the derived actor must not be the nil uuid")
	}
}

func TestActorFromSessionIsStableForTheSameUser(t *testing.T) {
	first, second := actorSeenBy(t, "user_01"), actorSeenBy(t, "user_01")

	if first != second {
		t.Errorf("the same user must always get the same actor, got %s and %s", first, second)
	}
}

func TestActorFromSessionGivesEachUserItsOwnActor(t *testing.T) {
	if actorSeenBy(t, "user_01") == actorSeenBy(t, "user_02") {
		t.Error("different users must get different actors")
	}
}

func TestActorFromSessionKeepsTheUserInTheContext(t *testing.T) {
	var seen string
	next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		seen = requestcontext.UserID(r.Context())
	})

	serveHandler(controllers.ActorFromSession(newErrs(), next), sessionRequest("user_01"))

	if seen != "user_01" {
		t.Errorf("user in context: got %q, want user_01", seen)
	}
}

func TestActorFromSessionIgnoresTheActorHeader(t *testing.T) {
	var seen uuid.UUID
	next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		seen = requestcontext.ActorID(r.Context())
	})
	req := sessionRequest("user_01")
	req.Header.Set(controllers.ActorIDHeader, uuid.NewString())

	serveHandler(controllers.ActorFromSession(newErrs(), next), req)

	if want := uuid.NewSHA1(uuid.NameSpaceURL, []byte("user_01")); seen != want {
		t.Errorf("actor in context: got %s, want %s", seen, want)
	}
}

func TestActorFromSessionWithoutUserIsUnauthorized(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true })

	rec := serveHandler(controllers.ActorFromSession(newErrs(), next), sessionRequest(""))

	if called {
		t.Error("the next handler must not run without a user")
	}
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status: got %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestRejectSessionMapsErrors(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		want     int
		wantCode string
		wantKind string
	}{
		{"unauthenticated", session.ErrUnauthenticated, http.StatusUnauthorized, string(exception.CodeUnauthorized), "unauthorized"},
		{"wrapped unauthenticated", fmt.Errorf("check: %w", session.ErrUnauthenticated), http.StatusUnauthorized, string(exception.CodeUnauthorized), "unauthorized"},
		{"unavailable", session.ErrAuthUnavailable, http.StatusServiceUnavailable, string(exception.CodeAuthUnavailable), "unavailable"},
		{"wrapped unavailable", fmt.Errorf("%w: jwks down", session.ErrAuthUnavailable), http.StatusServiceUnavailable, string(exception.CodeAuthUnavailable), "unavailable"},
		{"cookie too large", session.ErrCookieTooLarge, http.StatusInternalServerError, string(exception.CodeInternal), "internal"},
		{"unknown error", errors.New("boom"), http.StatusInternalServerError, string(exception.CodeInternal), "internal"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()

			controllers.RejectSession(newErrs())(rec, httptest.NewRequest(http.MethodGet, "/", nil), tt.err)

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
			if body["kind"] != tt.wantKind {
				t.Errorf("kind: got %q, want %q", body["kind"], tt.wantKind)
			}
			if body["error"] == "" {
				t.Error("error message must not be empty")
			}
		})
	}
}
