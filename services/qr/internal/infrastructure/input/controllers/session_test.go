package controllers_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"rutadeorigen/lib/requestcontext"
	"rutadeorigen/lib/session"
	"rutadeorigen/qr/internal/application/input/dto/request"
	"rutadeorigen/qr/internal/application/input/dto/response"
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

type actorResolverFake struct {
	byUser map[string]response.ResolvedActor
	err    error
	got    []request.ResolveActor
}

func (f *actorResolverFake) Execute(_ context.Context, in request.ResolveActor) (response.ResolvedActor, error) {
	f.got = append(f.got, in)
	if f.err != nil {
		return response.ResolvedActor{}, f.err
	}
	if actor, ok := f.byUser[in.UserID]; ok {
		return actor, nil
	}
	return response.ResolvedActor{}, exception.New(exception.CodeActorViewNotFound)
}

func knownActors(userID string, role string) (*actorResolverFake, uuid.UUID) {
	id := uuid.New()
	return &actorResolverFake{byUser: map[string]response.ResolvedActor{userID: {ActorID: id, Role: role}}}, id
}

func TestActorFromSessionTakesTheActorAndRoleFromTheProjection(t *testing.T) {
	fake, id := knownActors("user_01", "brand")
	var seenID uuid.UUID
	var seenRole, seenUser string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenID = requestcontext.ActorID(r.Context())
		seenRole = requestcontext.ActorRole(r.Context())
		seenUser = requestcontext.UserID(r.Context())
		w.WriteHeader(http.StatusNoContent)
	})

	rec := serveHandler(controllers.ActorFromSession(newErrs(), fake, next), sessionRequest("user_01"))

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status: got %d, want %d", rec.Code, http.StatusNoContent)
	}
	if seenID != id || seenRole != "brand" || seenUser != "user_01" {
		t.Errorf("got actor %s role %q user %q, want %s brand user_01", seenID, seenRole, seenUser, id)
	}
	if len(fake.got) != 1 || fake.got[0].UserID != "user_01" || fake.got[0].ActorID != "" {
		t.Errorf("unexpected resolver calls %+v", fake.got)
	}
}

func TestActorFromSessionIgnoresTheActorHeader(t *testing.T) {
	fake, id := knownActors("user_01", "producer")
	var seen uuid.UUID
	next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		seen = requestcontext.ActorID(r.Context())
	})
	req := sessionRequest("user_01")
	req.Header.Set(controllers.ActorIDHeader, uuid.NewString())

	serveHandler(controllers.ActorFromSession(newErrs(), fake, next), req)

	if seen != id {
		t.Errorf("actor in context: got %s, want %s", seen, id)
	}
}

func TestActorFromSessionRejectsAnUnknownActor(t *testing.T) {
	fake, _ := knownActors("user_01", "brand")
	called := false
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true })

	rec := serveHandler(controllers.ActorFromSession(newErrs(), fake, next), sessionRequest("user_99"))

	if called || rec.Code != http.StatusForbidden {
		t.Errorf("got status %d and called %v, want 403 and false", rec.Code, called)
	}
}

func TestActorFromSessionMapsResolverFailures(t *testing.T) {
	fake := &actorResolverFake{err: exception.New(exception.CodeInternal)}
	called := false
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true })

	rec := serveHandler(controllers.ActorFromSession(newErrs(), fake, next), sessionRequest("user_01"))

	if called || rec.Code != http.StatusInternalServerError {
		t.Errorf("got status %d and called %v, want 500 and false", rec.Code, called)
	}
}

func TestActorFromSessionWithoutUserIsUnauthorized(t *testing.T) {
	fake := &actorResolverFake{}
	called := false
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true })

	rec := serveHandler(controllers.ActorFromSession(newErrs(), fake, next), sessionRequest(""))

	if called || len(fake.got) != 0 {
		t.Error("neither the next handler nor the resolver must run without a user")
	}
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status: got %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

type headerResolverFake struct {
	role string
	err  error
	got  []request.ResolveActor
}

func (f *headerResolverFake) Execute(_ context.Context, in request.ResolveActor) (response.ResolvedActor, error) {
	f.got = append(f.got, in)
	if f.err != nil {
		return response.ResolvedActor{}, f.err
	}
	return response.ResolvedActor{Role: f.role}, nil
}

func headerRequest(actorID string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/public/lots/abc/qr", nil)
	if actorID != "" {
		req.Header.Set(controllers.ActorIDHeader, actorID)
	}
	return req
}

func TestRequireActorHeaderAddsTheRoleWhenTheActorIsKnown(t *testing.T) {
	id := uuid.New()
	fake := &headerResolverFake{role: "roaster"}
	var seenID uuid.UUID
	var seenRole string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenID, seenRole = requestcontext.ActorID(r.Context()), requestcontext.ActorRole(r.Context())
		w.WriteHeader(http.StatusNoContent)
	})

	rec := serveHandler(controllers.RequireActorHeader(newErrs(), fake, next), headerRequest(id.String()))

	if rec.Code != http.StatusNoContent || seenID != id || seenRole != "roaster" {
		t.Errorf("got status %d actor %s role %q", rec.Code, seenID, seenRole)
	}
	if len(fake.got) != 1 || fake.got[0].ActorID != id.String() || fake.got[0].UserID != "" {
		t.Errorf("unexpected resolver calls %+v", fake.got)
	}
}

func TestRequireActorHeaderKeepsTheActorWithoutRoleWhenUnknown(t *testing.T) {
	id := uuid.New()
	fake := &headerResolverFake{err: exception.New(exception.CodeActorViewNotFound)}
	var seenID uuid.UUID
	var seenRole string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenID, seenRole = requestcontext.ActorID(r.Context()), requestcontext.ActorRole(r.Context())
		w.WriteHeader(http.StatusNoContent)
	})

	rec := serveHandler(controllers.RequireActorHeader(newErrs(), fake, next), headerRequest(id.String()))

	if rec.Code != http.StatusNoContent || seenID != id || seenRole != "" {
		t.Errorf("got status %d actor %s role %q", rec.Code, seenID, seenRole)
	}
}

func TestRequireActorHeaderFailsOnResolverErrors(t *testing.T) {
	fake := &headerResolverFake{err: exception.New(exception.CodeInternal)}
	called := false
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true })

	rec := serveHandler(controllers.RequireActorHeader(newErrs(), fake, next), headerRequest(uuid.NewString()))

	if called || rec.Code != http.StatusInternalServerError {
		t.Errorf("got status %d and called %v, want 500 and false", rec.Code, called)
	}
}

func TestRequireActorHeaderRejectsAMissingOrInvalidHeader(t *testing.T) {
	for _, value := range []string{"", "not-a-uuid"} {
		fake := &headerResolverFake{}
		called := false
		next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true })

		rec := serveHandler(controllers.RequireActorHeader(newErrs(), fake, next), headerRequest(value))

		if called || len(fake.got) != 0 || rec.Code != http.StatusUnauthorized {
			t.Errorf("header %q: got status %d, called %v, resolver calls %d", value, rec.Code, called, len(fake.got))
		}
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
