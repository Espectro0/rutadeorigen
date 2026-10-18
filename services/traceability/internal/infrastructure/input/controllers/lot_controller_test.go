package controllers_test

import (
	"context"
	"encoding/json"
	"errors"
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

type fakeGetLotPort struct {
	result response.Lot
	err    error

	calls    int
	received request.GetLot
}

func (f *fakeGetLotPort) Execute(_ context.Context, input request.GetLot) (response.Lot, error) {
	f.calls++
	f.received = input
	if f.err != nil {
		return response.Lot{}, f.err
	}
	return f.result, nil
}

type fakeListLotsPort struct {
	result response.LotPage
	err    error

	calls    int
	received request.ListLots
}

func (f *fakeListLotsPort) Execute(_ context.Context, input request.ListLots) (response.LotPage, error) {
	f.calls++
	f.received = input
	if f.err != nil {
		return response.LotPage{}, f.err
	}
	return f.result, nil
}

type ctxKey struct{}

func passThrough(next http.Handler) http.Handler { return next }

func newMux(t *testing.T, port *fakeCreateLotPort, middleware controllers.Middleware) *http.ServeMux {
	t.Helper()

	mux := http.NewServeMux()
	controllers.NewLotController(port, &fakeGetLotPort{}, &fakeListLotsPort{}, newResponder(t), discardLogger()).Register(mux, middleware)
	return mux
}

func decodeBody(t *testing.T, rec *httptest.ResponseRecorder) map[string]string {
	t.Helper()

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("body is not JSON: %v", err)
	}
	return body
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

	rec := post(newMux(t, port, passThrough), validBody)

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

	post(newMux(t, port, markContext), validBody)

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

	rec := post(newMux(t, port, reject), validBody)

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

			rec := post(newMux(t, port, passThrough), tt.body)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("status: got %d, want %d", rec.Code, http.StatusBadRequest)
			}
			if port.calls != 0 {
				t.Error("an invalid body must not reach the port")
			}

			body := decodeBody(t, rec)
			if body["code"] != string(exception.CodeInvalidRequestBody) {
				t.Errorf("code: got %q, want %q", body["code"], exception.CodeInvalidRequestBody)
			}
			if body["error"] == "" {
				t.Error("error message must not be empty")
			}
		})
	}
}

func TestCreateLotMapsPortErrors(t *testing.T) {
	invalidCode := exception.New(exception.CodeLotInvalidCode).
		WithParams(map[string]any{"min": 3, "max": 30})

	tests := []struct {
		name     string
		err      error
		want     int
		wantCode string
	}{
		{"coded validation", invalidCode, http.StatusBadRequest, string(exception.CodeLotInvalidCode)},
		{"coded not found", exception.New(exception.CodeFarmNotFound), http.StatusNotFound, string(exception.CodeFarmNotFound)},
		{"coded conflict", exception.New(exception.CodeLotCodeAlreadyExists), http.StatusConflict, string(exception.CodeLotCodeAlreadyExists)},
		{"coded forbidden", exception.New(exception.CodeForbidden), http.StatusForbidden, string(exception.CodeForbidden)},
		{"internal", exception.New(exception.CodeInternal).WithTechnical("boom"), http.StatusInternalServerError, string(exception.CodeInternal)},
		{"plain error", errors.New("boom"), http.StatusInternalServerError, string(exception.CodeInternal)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := post(newMux(t, &fakeCreateLotPort{err: tt.err}, passThrough), validBody)

			if rec.Code != tt.want {
				t.Errorf("status: got %d, want %d", rec.Code, tt.want)
			}
			if rec.Header().Get("Location") != "" {
				t.Error("errors must not set the Location header")
			}

			body := decodeBody(t, rec)
			if body["code"] != tt.wantCode {
				t.Errorf("code: got %q, want %q", body["code"], tt.wantCode)
			}
			if body["error"] == "" {
				t.Error("error message must not be empty")
			}
		})
	}
}

func TestLotsCollectionRejectsUnsupportedMethods(t *testing.T) {
	for _, method := range []string{http.MethodPut, http.MethodPatch, http.MethodDelete} {
		t.Run(method, func(t *testing.T) {
			port := &fakeCreateLotPort{}
			req := httptest.NewRequest(method, "/lots", nil)
			rec := httptest.NewRecorder()

			newMux(t, port, passThrough).ServeHTTP(rec, req)

			if rec.Code != http.StatusMethodNotAllowed || port.calls != 0 {
				t.Errorf("got status %d and %d port calls, want 405 and 0", rec.Code, port.calls)
			}
		})
	}
}
