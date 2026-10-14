package middleware_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"rutadeorigen/lib/httpserver/middleware"
	"rutadeorigen/lib/logger"
)

func accessLogEntries(t *testing.T, buf *bytes.Buffer) []map[string]any {
	t.Helper()

	var entries []map[string]any
	dec := json.NewDecoder(buf)
	for dec.More() {
		var entry map[string]any
		if err := dec.Decode(&entry); err != nil {
			t.Fatalf("log line is not valid JSON: %v", err)
		}
		entries = append(entries, entry)
	}
	return entries
}

func singleEntry(t *testing.T, buf *bytes.Buffer) map[string]any {
	t.Helper()

	entries := accessLogEntries(t, buf)
	if len(entries) != 1 {
		t.Fatalf("expected 1 log line, got %d: %v", len(entries), entries)
	}
	return entries[0]
}

func TestAccessLog_LevelDependsOnStatus(t *testing.T) {
	tests := []struct {
		name      string
		status    int
		wantLevel string
	}{
		{"ok", http.StatusOK, "INFO"},
		{"created", http.StatusCreated, "INFO"},
		{"bad request", http.StatusBadRequest, "WARN"},
		{"not found", http.StatusNotFound, "WARN"},
		{"internal", http.StatusInternalServerError, "ERROR"},
		{"unavailable", http.StatusServiceUnavailable, "ERROR"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.status)
			})
			rec := httptest.NewRecorder()

			middleware.AccessLog(logger.New(&buf, "test", slog.LevelDebug), next).
				ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/lots", nil))

			if rec.Code != tt.status {
				t.Errorf("status must reach the client: got %d, want %d", rec.Code, tt.status)
			}
			entry := singleEntry(t, &buf)
			if entry["level"] != tt.wantLevel {
				t.Errorf("level = %v, expected %s", entry["level"], tt.wantLevel)
			}
			if entry["http.response.status_code"] != float64(tt.status) {
				t.Errorf("http.response.status_code = %v, expected %d", entry["http.response.status_code"], tt.status)
			}
		})
	}
}

func TestAccessLog_DefaultsToOKWhenHeaderIsNotWritten(t *testing.T) {
	var buf bytes.Buffer
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("pong"))
	})

	middleware.AccessLog(logger.New(&buf, "test", slog.LevelInfo), next).
		ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/ping", nil))

	entry := singleEntry(t, &buf)
	if entry["http.response.status_code"] != float64(http.StatusOK) || entry["level"] != "INFO" {
		t.Errorf("expected an INFO line with status 200, got %v", entry)
	}
}

func TestAccessLog_UsesSemanticConventionFields(t *testing.T) {
	var buf bytes.Buffer
	mux := http.NewServeMux()
	mux.HandleFunc("POST /lots/{id}", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})

	middleware.AccessLog(logger.New(&buf, "test", slog.LevelInfo), mux).
		ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/lots/42", nil))

	entry := singleEntry(t, &buf)
	want := map[string]any{
		"msg":                       "http request",
		"http.request.method":       "POST",
		"url.path":                  "/lots/42",
		"http.route":                "POST /lots/{id}",
		"http.response.status_code": float64(http.StatusCreated),
	}
	for k, v := range want {
		if entry[k] != v {
			t.Errorf("%s = %v, expected %v", k, entry[k], v)
		}
	}

	duration, ok := entry["http.server.request.duration"].(float64)
	if !ok || duration < 0 {
		t.Errorf("http.server.request.duration must be a non-negative number of seconds, got %v", entry["http.server.request.duration"])
	}
}

func TestAccessLog_SkipsHealthChecks(t *testing.T) {
	for _, path := range []string{"/healthz", "/readyz"} {
		t.Run(path, func(t *testing.T) {
			var buf bytes.Buffer
			called := false
			next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				called = true
				w.WriteHeader(http.StatusOK)
			})

			middleware.AccessLog(logger.New(&buf, "test", slog.LevelDebug), next).
				ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, path, nil))

			if !called {
				t.Error("health checks must still reach the handler")
			}
			if buf.Len() != 0 {
				t.Errorf("health checks must not be logged, got %s", buf.String())
			}
		})
	}
}

func TestAccessLog_IncludesCorrelationIDWhenWrappedByCorrelation(t *testing.T) {
	var buf bytes.Buffer
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	req := httptest.NewRequest(http.MethodGet, "/lots", nil)
	req.Header.Set(middleware.CorrelationIDHeader, "req-abc")

	middleware.Correlation(middleware.AccessLog(logger.New(&buf, "test", slog.LevelInfo), next)).
		ServeHTTP(httptest.NewRecorder(), req)

	entry := singleEntry(t, &buf)
	if entry["correlation_id"] != "req-abc" {
		t.Errorf("correlation_id = %v, expected req-abc", entry["correlation_id"])
	}
}
