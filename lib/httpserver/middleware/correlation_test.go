package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"rutadeorigen/lib/httpserver/middleware"
	"rutadeorigen/lib/requestcontext"
)

func serve(t *testing.T, headerValue string, setHeader bool) (seen string, rec *httptest.ResponseRecorder) {
	t.Helper()

	next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		seen = requestcontext.CorrelationID(r.Context())
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if setHeader {
		req.Header.Set(middleware.CorrelationIDHeader, headerValue)
	}
	rec = httptest.NewRecorder()

	middleware.Correlation(next).ServeHTTP(rec, req)
	return seen, rec
}

func TestCorrelationKeepsValidIncomingID(t *testing.T) {
	valid := []string{
		"req-123",
		"ABC_def-789",
		uuid.NewString(),
		strings.Repeat("a", 64),
	}

	for _, id := range valid {
		t.Run(id, func(t *testing.T) {
			seen, rec := serve(t, id, true)

			if seen != id {
				t.Errorf("context: got %q, want %q", seen, id)
			}
			if got := rec.Header().Get(middleware.CorrelationIDHeader); got != id {
				t.Errorf("response header: got %q, want %q", got, id)
			}
		})
	}
}

func TestCorrelationGeneratesIDWhenMissingOrInvalid(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		setHeader bool
	}{
		{"header missing", "", false},
		{"header empty", "", true},
		{"too long", strings.Repeat("a", 65), true},
		{"contains spaces", "req 123", true},
		{"contains newline", "req\n123", true},
		{"contains special characters", "req;drop", true},
		{"contains non ascii", "pedido-ñ", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			seen, rec := serve(t, tt.value, tt.setHeader)

			if _, err := uuid.Parse(seen); err != nil {
				t.Errorf("expected a generated UUID in the context, got %q", seen)
			}
			if seen == tt.value {
				t.Errorf("invalid value %q must not be propagated", tt.value)
			}
			if got := rec.Header().Get(middleware.CorrelationIDHeader); got != seen {
				t.Errorf("response header %q must match the context value %q", got, seen)
			}
		})
	}
}

func TestCorrelationGeneratesDifferentIDs(t *testing.T) {
	first, _ := serve(t, "", false)
	second, _ := serve(t, "", false)

	if first == second {
		t.Errorf("each request must get its own ID, both got %q", first)
	}
}
