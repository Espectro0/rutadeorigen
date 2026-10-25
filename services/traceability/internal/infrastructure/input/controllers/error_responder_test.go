package controllers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"rutadeorigen/lib/logger"
	"rutadeorigen/lib/requestcontext"
	"rutadeorigen/traceability/internal/platform/catalog"
	"rutadeorigen/traceability/internal/platform/exception"
)

type fakeResolver struct {
	err       error
	calls     int
	gotCode   string
	gotParams map[string]any
}

func (f *fakeResolver) Resolve(code string, params map[string]any) (string, error) {
	f.calls++
	f.gotCode = code
	f.gotParams = params

	if f.err != nil {
		return "fallback message", f.err
	}
	return "message for " + code, nil
}

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

func TestWriteMapsCodedExceptionsToStatusAndMessage(t *testing.T) {
	tests := []struct {
		name       string
		code       exception.Code
		wantStatus int
		wantKind   string
	}{
		{"validation", exception.CodeLotInvalidVariety, http.StatusBadRequest, "validation"},
		{"not found", exception.CodeActorNotFound, http.StatusNotFound, "not_found"},
		{"forbidden", exception.CodeForbidden, http.StatusForbidden, "forbidden"},
		{"unauthorized", exception.CodeUnauthorized, http.StatusUnauthorized, "unauthorized"},
		{"conflict", exception.CodeLotVoided, http.StatusConflict, "conflict"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var logs bytes.Buffer
			rec := httptest.NewRecorder()
			responder := NewErrorResponder(slog.New(slog.NewJSONHandler(&logs, nil)), &fakeResolver{})

			responder.Write(rec, newRequest("req-123"), exception.New(tt.code))

			if rec.Code != tt.wantStatus {
				t.Errorf("status: got %d, want %d", rec.Code, tt.wantStatus)
			}
			body := decodeError(t, rec)
			if body.Kind != tt.wantKind {
				t.Errorf("kind: got %q, want %q", body.Kind, tt.wantKind)
			}
			if body.Code != string(tt.code) {
				t.Errorf("code: got %q, want %q", body.Code, tt.code)
			}
			if want := "message for " + string(tt.code); body.Error != want {
				t.Errorf("error: got %q, want %q", body.Error, want)
			}
			if body.CorrelationID != "req-123" {
				t.Errorf("correlation id: got %q, want req-123", body.CorrelationID)
			}
			out := logs.String()
			if !strings.Contains(out, `"level":"WARN"`) || !strings.Contains(out, string(tt.code)) {
				t.Errorf("client errors must be logged as WARN with the code, got %s", out)
			}
			if strings.Contains(out, `"level":"ERROR"`) {
				t.Errorf("client errors must not be logged as ERROR, got %s", out)
			}
		})
	}
}

func TestWritePassesCodeAndParamsToTheResolver(t *testing.T) {
	resolver := &fakeResolver{}
	responder := NewErrorResponder(slog.New(slog.DiscardHandler), resolver)
	err := exception.New(exception.CodeLotInvalidCode).WithParams(map[string]any{"min": 3, "max": 30})

	responder.Write(httptest.NewRecorder(), newRequest(""), err)

	if resolver.gotCode != string(exception.CodeLotInvalidCode) {
		t.Errorf("code: got %q, want %q", resolver.gotCode, exception.CodeLotInvalidCode)
	}
	if resolver.gotParams["min"] != 3 || resolver.gotParams["max"] != 30 {
		t.Errorf("params: got %v", resolver.gotParams)
	}
}

func TestWriteKeepsCodeOfWrappedException(t *testing.T) {
	coded := exception.New(exception.CodeActorNotFound)
	wrapped := map[string]error{
		"errors.Join": errors.Join(errors.New("context"), coded),
		"fmt.Errorf":  fmt.Errorf("find actor: %w", coded),
	}

	for name, err := range wrapped {
		t.Run(name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			responder := NewErrorResponder(slog.New(slog.DiscardHandler), &fakeResolver{})

			responder.Write(rec, newRequest(""), err)

			if rec.Code != http.StatusNotFound {
				t.Errorf("status: got %d, want %d", rec.Code, http.StatusNotFound)
			}
			if body := decodeError(t, rec); body.Code != string(exception.CodeActorNotFound) {
				t.Errorf("code: got %q, want %q", body.Code, exception.CodeActorNotFound)
			}
		})
	}
}

func TestWriteIncludesCodeInTheJSONBody(t *testing.T) {
	rec := httptest.NewRecorder()
	responder := NewErrorResponder(slog.New(slog.DiscardHandler), &fakeResolver{})

	responder.Write(rec, newRequest(""), exception.New(exception.CodeFarmNotFound))

	want := `"code":"` + string(exception.CodeFarmNotFound) + `"`
	if !strings.Contains(rec.Body.String(), want) {
		t.Errorf("body %s does not contain %s", rec.Body.String(), want)
	}
}

func TestWriteHidesInternalDetails(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{"internal exception with cause", exception.New(exception.CodeInternal).WithTechnical("db is down").WithCause(errors.New("connection refused"))},
		{"internal exception", exception.New(exception.CodeInternal)},
		{"plain error", errors.New("connection refused")},
		{"unregistered code", exception.New("NOT_A_REAL_CODE")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var logs bytes.Buffer
			resolver := &fakeResolver{}
			rec := httptest.NewRecorder()
			responder := NewErrorResponder(logger.New(&logs, "test", slog.LevelInfo), resolver)

			responder.Write(rec, newRequest("req-500"), tt.err)

			if rec.Code != http.StatusInternalServerError {
				t.Errorf("status: got %d, want %d", rec.Code, http.StatusInternalServerError)
			}
			body := decodeError(t, rec)
			if body.Code != string(exception.CodeInternal) || body.Kind != "internal" || body.CorrelationID != "req-500" {
				t.Errorf("unexpected body: %+v", body)
			}
			if want := "message for " + string(exception.CodeInternal); body.Error != want {
				t.Errorf("error: got %q, want the INTERNAL message %q", body.Error, want)
			}
			if resolver.gotCode != string(exception.CodeInternal) {
				t.Errorf("resolved code: got %q, want %q", resolver.gotCode, exception.CodeInternal)
			}
			if strings.Contains(rec.Body.String(), "connection refused") || strings.Contains(rec.Body.String(), "db is down") {
				t.Errorf("internal details leaked to the client: %s", rec.Body.String())
			}
			if !strings.Contains(logs.String(), "request failed") || !strings.Contains(logs.String(), `"correlation_id":"req-500"`) {
				t.Errorf("internal errors must be logged with the correlation id, got %s", logs.String())
			}
			if strings.Contains(logs.String(), "connection refused") {
				t.Errorf("the technical cause must only be logged at DEBUG, got %s", logs.String())
			}
		})
	}
}

func TestWriteLogsTheCauseAtDebug(t *testing.T) {
	var logs bytes.Buffer
	responder := NewErrorResponder(logger.New(&logs, "test", slog.LevelDebug), &fakeResolver{})
	err := exception.New(exception.CodeInternal).WithCause(errors.New("connection refused"))

	responder.Write(httptest.NewRecorder(), newRequest("req-9"), err)

	out := logs.String()
	if !strings.Contains(out, "connection refused") || !strings.Contains(out, `"level":"DEBUG"`) {
		t.Errorf("the cause must be logged at DEBUG, got %s", out)
	}
}

func TestWriteLogsWhenTheMessageCannotBeResolved(t *testing.T) {
	var logs bytes.Buffer
	resolver := &fakeResolver{err: errors.New("missing parameter")}
	rec := httptest.NewRecorder()
	responder := NewErrorResponder(logger.New(&logs, "test", slog.LevelInfo), resolver)

	responder.Write(rec, newRequest("req-1"), exception.New(exception.CodeActorNotFound))

	if rec.Code != http.StatusNotFound {
		t.Errorf("status must be kept: got %d, want %d", rec.Code, http.StatusNotFound)
	}
	body := decodeError(t, rec)
	if body.Error != "fallback message" {
		t.Errorf("error: got %q, want the fallback message", body.Error)
	}
	if body.Code != string(exception.CodeActorNotFound) {
		t.Errorf("code: got %q, want %q", body.Code, exception.CodeActorNotFound)
	}

	out := logs.String()
	for _, want := range []string{
		"cannot resolve error message",
		`"level":"ERROR"`,
		string(exception.CodeActorNotFound),
		`"correlation_id":"req-1"`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("log %s does not contain %s", out, want)
		}
	}
}

func TestWriteWithoutCorrelationID(t *testing.T) {
	tests := map[string]error{
		"coded": exception.New(exception.CodeLotInvalidVariety),
	}

	for name, err := range tests {
		t.Run(name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			responder := NewErrorResponder(slog.New(slog.DiscardHandler), &fakeResolver{})

			responder.Write(rec, newRequest(""), err)

			if strings.Contains(rec.Body.String(), "correlation_id") {
				t.Errorf("empty correlation id must be omitted, got %s", rec.Body.String())
			}
		})
	}
}

func TestWriteWithTheRealCatalog(t *testing.T) {
	messages, err := catalog.New()
	if err != nil {
		t.Fatalf("catalog.New: %v", err)
	}
	responder := NewErrorResponder(slog.New(slog.DiscardHandler), messages)

	t.Run("message with parameters", func(t *testing.T) {
		rec := httptest.NewRecorder()
		ex := exception.New(exception.CodeLotInvalidCode).WithParams(map[string]any{"min": 3, "max": 30})

		responder.Write(rec, newRequest(""), ex)

		body := decodeError(t, rec)
		want := "El código del lote debe tener entre 3 y 30 caracteres."
		if rec.Code != http.StatusBadRequest || body.Error != want {
			t.Errorf("got status %d and error %q, want 400 and %q", rec.Code, body.Error, want)
		}
	})

	t.Run("internal errors use the INTERNAL message", func(t *testing.T) {
		rec := httptest.NewRecorder()
		want, _ := messages.Resolve(string(exception.CodeInternal), nil)

		responder.Write(rec, newRequest(""), errors.New("boom"))

		body := decodeError(t, rec)
		if rec.Code != http.StatusInternalServerError || body.Error != want {
			t.Errorf("got status %d and error %q, want 500 and %q", rec.Code, body.Error, want)
		}
	})
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
