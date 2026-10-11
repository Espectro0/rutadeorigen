package controllers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"rutadeorigen/traceability/internal/application/input/dto/request"
	"rutadeorigen/traceability/internal/application/input/dto/response"
	"rutadeorigen/traceability/internal/infrastructure/input/controllers"
	"rutadeorigen/traceability/internal/platform/exception"
)

const validBody = `{"farm_id":"22222222-2222-2222-2222-222222222222","code":"LOT-001","variety":"Castillo","process":"washed","harvest_date":"2026-09-15"}`

type fakeCreateLotPort struct {
	result response.Lot
	err    error

	calls    int
	received request.CreateLot
	ctx      context.Context
}

func (f *fakeCreateLotPort) Execute(ctx context.Context, input request.CreateLot) (response.Lot, error) {
	f.calls++
	f.received = input
	f.ctx = ctx
	if f.err != nil {
		return response.Lot{}, f.err
	}
	return f.result, nil
}

type ctxKey struct{}

func passThrough(next http.Handler) http.Handler { return next }

func newMux(port *fakeCreateLotPort, middleware controllers.Middleware) *http.ServeMux {
	mux := http.NewServeMux()
	controllers.NewLotController(port, discardLogger()).Register(mux, middleware)
	return mux
}

func post(mux http.Handler, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/lots", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func TestCreateLotReturnsCreated(t *testing.T) {
	port := &fakeCreateLotPort{result: response.Lot{ID: "lot-1", Code: "LOT-001", Status: "active"}}

	rec := post(newMux(port, passThrough), validBody)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status: got %d, want %d (%s)", rec.Code, http.StatusCreated, rec.Body.String())
	}
	if got := rec.Header().Get("Location"); got != "/lots/lot-1" {
		t.Errorf("Location: got %q, want /lots/lot-1", got)
	}

	var body response.Lot
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("body is not JSON: %v", err)
	}
	if body != port.result {
		t.Errorf("body: got %+v, want %+v", body, port.result)
	}

	want := request.CreateLot{
		FarmID:      "22222222-2222-2222-2222-222222222222",
		Code:        "LOT-001",
		Variety:     "Castillo",
		Process:     "washed",
		HarvestDate: "2026-09-15",
	}
	if port.received != want {
		t.Errorf("port received %+v, want %+v", port.received, want)
	}
}

func TestCreateLotRunsTheMiddleware(t *testing.T) {
	port := &fakeCreateLotPort{}
	markContext := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, "marked")))
		})
	}

	post(newMux(port, markContext), validBody)

	if port.calls != 1 || port.ctx.Value(ctxKey{}) != "marked" {
		t.Error("the request context from the middleware must reach the port")
	}
}

func TestCreateLotStopsWhenMiddlewareRejects(t *testing.T) {
	port := &fakeCreateLotPort{}
	reject := func(http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		})
	}

	rec := post(newMux(port, reject), validBody)

	if rec.Code != http.StatusUnauthorized || port.calls != 0 {
		t.Errorf("got status %d and %d port calls, want 401 and 0", rec.Code, port.calls)
	}
}

func TestCreateLotRejectsInvalidBody(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{"empty body", ""},
		{"malformed json", `{"farm_id":`},
		{"not an object", `["LOT-001"]`},
		{"unknown field", `{"farm_id":"x","farmId":"y"}`},
		{"wrong field type", `{"code":123}`},
		{"body too large", `{"code":"` + strings.Repeat("a", 1<<20) + `"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			port := &fakeCreateLotPort{}

			rec := post(newMux(port, passThrough), tt.body)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("status: got %d, want %d", rec.Code, http.StatusBadRequest)
			}
			if port.calls != 0 {
				t.Error("an invalid body must not reach the port")
			}
		})
	}
}

func TestCreateLotMapsPortErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{"validation", exception.NewValidation(exception.LayerApplication, "Dato inválido", "bad"), http.StatusBadRequest},
		{"not found", exception.NewNotFound(exception.LayerApplication, "No existe", "missing"), http.StatusNotFound},
		{"forbidden", exception.NewForbidden(exception.LayerApplication, "No permitido", "denied"), http.StatusForbidden},
		{"conflict", exception.NewConflict(exception.LayerApplication, "Ya existe", "duplicated"), http.StatusConflict},
		{"internal", exception.NewInternal(exception.LayerInfrastructure, "", "boom", nil), http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := post(newMux(&fakeCreateLotPort{err: tt.err}, passThrough), validBody)

			if rec.Code != tt.want {
				t.Errorf("status: got %d, want %d", rec.Code, tt.want)
			}
			if rec.Header().Get("Location") != "" {
				t.Error("errors must not set the Location header")
			}
		})
	}
}

func TestCreateLotOnlyAcceptsPost(t *testing.T) {
	port := &fakeCreateLotPort{}
	req := httptest.NewRequest(http.MethodGet, "/lots", nil)
	rec := httptest.NewRecorder()

	newMux(port, passThrough).ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed || port.calls != 0 {
		t.Errorf("got status %d and %d port calls, want 405 and 0", rec.Code, port.calls)
	}
}
