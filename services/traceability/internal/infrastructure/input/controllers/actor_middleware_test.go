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
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
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

	controllers.RequireActor(discardLogger(), next).ServeHTTP(rec, req)

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

			controllers.RequireActor(discardLogger(), next).ServeHTTP(rec, req)

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
			if body["kind"] != "unauthorized" || body["error"] == "" {
				t.Errorf("unexpected body: %v", body)
			}
		})
	}
}
