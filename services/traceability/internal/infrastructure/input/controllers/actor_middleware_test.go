package controllers_test

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
	"rutadeorigen/traceability/internal/application/input/dto/request"
	"rutadeorigen/traceability/internal/application/input/dto/response"
	"rutadeorigen/traceability/internal/infrastructure/input/controllers"
	"rutadeorigen/traceability/internal/platform/catalog"
	"rutadeorigen/traceability/internal/platform/exception"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func newResponder(t *testing.T) *controllers.ErrorResponder {
	t.Helper()

	messages, err := catalog.New()
	if err != nil {
		t.Fatalf("catalog.New: %v", err)
	}
	return controllers.NewErrorResponder(discardLogger(), messages)
}

type fakeResolver struct {
	result response.ResolvedActor
	err    error

	calls    int
	received string
}

func (f *fakeResolver) Execute(ctx context.Context, _ request.ResolveActor) (response.ResolvedActor, error) {
	f.calls++
	f.received = requestcontext.UserID(ctx)
	if f.err != nil {
		return response.ResolvedActor{}, f.err
	}
	return f.result, nil
}

func resolvedActor() response.ResolvedActor {
	return response.ResolvedActor{ID: uuid.New()}
}

func sessionRequest(userID string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/lots", nil)
	return req.WithContext(requestcontext.WithUserID(req.Context(), userID))
}

func TestRequireActorHeaderStoresTheActorInTheContext(t *testing.T) {
	actorID := uuid.New()
	var seen uuid.UUID
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = requestcontext.ActorID(r.Context())
		w.WriteHeader(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodPost, "/lots", nil)
	req.Header.Set(controllers.ActorIDHeader, actorID.String())
	rec := httptest.NewRecorder()

	controllers.RequireActorHeader(newResponder(t), next).ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status: got %d, want %d", rec.Code, http.StatusNoContent)
	}
	if seen != actorID {
		t.Errorf("actor in context: got %s, want %s", seen, actorID)
	}
}

func TestRequireActorHeaderRejectsMissingOrInvalidActor(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		setHeader bool
	}{
		{"header missing", "", false},
		{"header empty", "", true},
		{"not a uuid", "producer-1", true},
		{"nil uuid", uuid.Nil.String(), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true })

			req := httptest.NewRequest(http.MethodPost, "/lots", nil)
			if tt.setHeader {
				req.Header.Set(controllers.ActorIDHeader, tt.value)
			}
			rec := httptest.NewRecorder()

			controllers.RequireActorHeader(newResponder(t), next).ServeHTTP(rec, req)

			if called {
				t.Error("the next handler must not run without a valid actor")
			}
			if rec.Code != http.StatusUnauthorized {
				t.Errorf("status: got %d, want %d", rec.Code, http.StatusUnauthorized)
			}

			var body map[string]string
			if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
				t.Fatalf("body is not JSON: %v", err)
			}
			if body["kind"] != "unauthorized" {
				t.Errorf("kind: got %q, want unauthorized", body["kind"])
			}
			if body["code"] != string(exception.CodeUnauthorized) {
				t.Errorf("code: got %q, want %q", body["code"], exception.CodeUnauthorized)
			}
			if body["error"] == "" {
				t.Error("error message must not be empty")
			}
		})
	}
}

func TestRequireActorStoresTheResolvedActorInTheContext(t *testing.T) {
	resolved := resolvedActor()
	resolver := &fakeResolver{result: resolved}
	var seenActor uuid.UUID
	var seenUser string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenActor = requestcontext.ActorID(r.Context())
		seenUser = requestcontext.UserID(r.Context())
		w.WriteHeader(http.StatusNoContent)
	})
	rec := httptest.NewRecorder()

	controllers.RequireActor(resolver, newResponder(t), next).ServeHTTP(rec, sessionRequest("user_01"))

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status: got %d, want %d", rec.Code, http.StatusNoContent)
	}
	if resolver.calls != 1 || resolver.received != "user_01" {
		t.Errorf("resolver got %d calls with %q, want 1 call with user_01", resolver.calls, resolver.received)
	}
	if seenActor != resolved.ID {
		t.Errorf("actor in context: got %s, want %s", seenActor, resolved.ID)
	}
	if seenUser != "user_01" {
		t.Errorf("user in context: got %q, want user_01", seenUser)
	}
}

func TestRequireActorIgnoresTheActorHeader(t *testing.T) {
	resolved := resolvedActor()
	resolver := &fakeResolver{result: resolved}
	var seen uuid.UUID
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = requestcontext.ActorID(r.Context())
		w.WriteHeader(http.StatusNoContent)
	})
	req := sessionRequest("user_01")
	req.Header.Set(controllers.ActorIDHeader, uuid.NewString())
	rec := httptest.NewRecorder()

	controllers.RequireActor(resolver, newResponder(t), next).ServeHTTP(rec, req)

	if seen != resolved.ID {
		t.Errorf("actor in context: got %s, want the resolved actor %s", seen, resolved.ID)
	}
}

func TestRequireActorWithoutSessionIsUnauthorized(t *testing.T) {
	resolver := &fakeResolver{err: exception.New(exception.CodeUnauthorized)}
	called := false
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true })
	rec := httptest.NewRecorder()

	controllers.RequireActor(resolver, newResponder(t), next).ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/lots", nil))

	if called {
		t.Error("the next handler must not run without a session")
	}
	if resolver.received != "" {
		t.Errorf("resolver received %q, want empty", resolver.received)
	}
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status: got %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestRequireActorMapsResolverErrors(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		want     int
		wantCode string
		wantKind string
	}{
		{"unauthorized", exception.New(exception.CodeUnauthorized), http.StatusUnauthorized, string(exception.CodeUnauthorized), "unauthorized"},
		{"not registered", exception.New(exception.CodeActorNotFound), http.StatusNotFound, string(exception.CodeActorNotFound), "not_found"},
		{"inactive", exception.New(exception.CodeForbidden), http.StatusForbidden, string(exception.CodeForbidden), "forbidden"},
		{"internal", exception.New(exception.CodeInternal).WithTechnical("db password leaked"), http.StatusInternalServerError, string(exception.CodeInternal), "internal"},
		{"plain error", errors.New("boom"), http.StatusInternalServerError, string(exception.CodeInternal), "internal"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver := &fakeResolver{err: tt.err}
			called := false
			next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true })
			rec := httptest.NewRecorder()

			controllers.RequireActor(resolver, newResponder(t), next).ServeHTTP(rec, sessionRequest("user_01"))

			if called {
				t.Error("the next handler must not run when the actor cannot be resolved")
			}
			if rec.Code != tt.want {
				t.Errorf("status: got %d, want %d", rec.Code, tt.want)
			}

			raw := rec.Body.String()
			var body map[string]string
			if err := json.Unmarshal([]byte(raw), &body); err != nil {
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
			if strings.Contains(raw, "leaked") {
				t.Error("internal details must not reach the client")
			}
		})
	}
}
