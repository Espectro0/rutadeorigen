package controllers_test

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"rutadeorigen/lib/requestcontext"
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

func TestRequireActorStoresTheActorInTheContext(t *testing.T) {
	actorID := uuid.New()
	var seen uuid.UUID
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = requestcontext.ActorID(r.Context())
		w.WriteHeader(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodPost, "/lots", nil)
	req.Header.Set(controllers.ActorIDHeader, actorID.String())
	rec := httptest.NewRecorder()

	controllers.RequireActor(newResponder(t), next).ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status: got %d, want %d", rec.Code, http.StatusNoContent)
	}
	if seen != actorID {
		t.Errorf("actor in context: got %s, want %s", seen, actorID)
	}
}

func TestRequireActorRejectsMissingOrInvalidActor(t *testing.T) {
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

			controllers.RequireActor(newResponder(t), next).ServeHTTP(rec, req)

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
