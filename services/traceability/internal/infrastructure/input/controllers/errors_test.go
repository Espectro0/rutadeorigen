package controllers

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"rutadeorigen/lib/requestcontext"
	"rutadeorigen/traceability/internal/platform/exception"
)

func newRequest(correlationID string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/lots", nil)
	if correlationID != "" {
		req = req.WithContext(requestcontext.WithCorrelationID(req.Context(), correlationID))
	}
	return req
}

func decodeError(t *testing.T, rec *httptest.ResponseRecorder) errorResponse {
	t.Helper()

	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type: got %q, want application/json", got)
	}
	var body errorResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("body is not JSON: %v", err)
	}
	return body
}

func TestWriteErrorMapsKindsToStatus(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantKind   string
		wantUser   string
	}{
		{"validation", exception.NewValidation(exception.LayerApplication, "Dato inválido", "bad"), http.StatusBadRequest, "validation", "Dato inválido"},
		{"not found", exception.NewNotFound(exception.LayerApplication, "No existe", "missing"), http.StatusNotFound, "not_found", "No existe"},
		{"forbidden", exception.NewForbidden(exception.LayerApplication, "No permitido", "denied"), http.StatusForbidden, "forbidden", "No permitido"},
		{"unauthorized", exception.NewUnauthorized(exception.LayerInfrastructure, "Identifícate", "no actor"), http.StatusUnauthorized, "unauthorized", "Identifícate"},
		{"conflict", exception.NewConflict(exception.LayerApplication, "Ya existe", "duplicated"), http.StatusConflict, "conflict", "Ya existe"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var logs bytes.Buffer
			rec := httptest.NewRecorder()

			writeError(rec, newRequest("req-123"), slog.New(slog.NewJSONHandler(&logs, nil)), tt.err)

			if rec.Code != tt.wantStatus {
				t.Errorf("status: got %d, want %d", rec.Code, tt.wantStatus)
			}
			body := decodeError(t, rec)
			if body.Kind != tt.wantKind || body.Error != tt.wantUser || body.CorrelationID != "req-123" {
				t.Errorf("unexpected body: %+v", body)
			}
			if logs.Len() != 0 {
				t.Errorf("client errors must not be logged, got %s", logs.String())
			}
		})
	}
}

func TestWriteErrorKeepsKindOfWrappedException(t *testing.T) {
	wrapped := errors.Join(errors.New("context"), exception.NewNotFound(exception.LayerApplication, "No existe", "missing"))
	rec := httptest.NewRecorder()

	writeError(rec, newRequest(""), slog.New(slog.DiscardHandler), wrapped)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status: got %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestWriteErrorHidesInternalDetails(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{"internal exception", exception.NewInternal(exception.LayerInfrastructure, "Detalle interno", "db is down", errors.New("connection refused"))},
		{"plain error", errors.New("connection refused")},
		{"unknown kind", exception.New(exception.Kind("teapot"), exception.LayerDomain, "Detalle interno", "weird kind", nil)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var logs bytes.Buffer
			rec := httptest.NewRecorder()

			writeError(rec, newRequest("req-500"), slog.New(slog.NewJSONHandler(&logs, nil)), tt.err)

			if rec.Code != http.StatusInternalServerError {
				t.Errorf("status: got %d, want %d", rec.Code, http.StatusInternalServerError)
			}
			body := decodeError(t, rec)
			if body.Error != genericErrorMessage || body.Kind != "internal" || body.CorrelationID != "req-500" {
				t.Errorf("unexpected body: %+v", body)
			}
			if strings.Contains(rec.Body.String(), "connection refused") || strings.Contains(rec.Body.String(), "Detalle interno") {
				t.Errorf("internal details leaked to the client: %s", rec.Body.String())
			}
			if !strings.Contains(logs.String(), "request failed") || !strings.Contains(logs.String(), "req-500") {
				t.Errorf("internal errors must be logged with the correlation id, got %s", logs.String())
			}
		})
	}
}

func TestWriteErrorWithoutCorrelationID(t *testing.T) {
	rec := httptest.NewRecorder()

	writeError(rec, newRequest(""), slog.New(slog.DiscardHandler),
		exception.NewValidation(exception.LayerApplication, "Dato inválido", "bad"))

	if strings.Contains(rec.Body.String(), "correlation_id") {
		t.Errorf("empty correlation id must be omitted, got %s", rec.Body.String())
	}
}

func TestWriteJSON(t *testing.T) {
	t.Run("writes status, header and body", func(t *testing.T) {
		rec := httptest.NewRecorder()

		writeJSON(rec, slog.New(slog.DiscardHandler), http.StatusCreated, map[string]string{"id": "1"})

		if rec.Code != http.StatusCreated {
			t.Errorf("status: got %d, want %d", rec.Code, http.StatusCreated)
		}
		if got := rec.Header().Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type: got %q", got)
		}
		if got := strings.TrimSpace(rec.Body.String()); got != `{"id":"1"}` {
			t.Errorf("body: got %s", got)
		}
	})

	t.Run("logs when the body cannot be encoded", func(t *testing.T) {
		var logs bytes.Buffer
		rec := httptest.NewRecorder()

		writeJSON(rec, slog.New(slog.NewJSONHandler(&logs, nil)), http.StatusOK, make(chan int))

		if logs.Len() == 0 {
			t.Error("encoding errors must be logged")
		}
	})
}
