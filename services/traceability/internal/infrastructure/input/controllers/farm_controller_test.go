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

const validFarmBody = `{"name":"La Esperanza","altitude":1650}`

type fakeCreateFarmPort struct {
	result response.Farm
	err    error

	calls    int
	received request.CreateFarm
	ctx      context.Context
}

func (f *fakeCreateFarmPort) Execute(ctx context.Context, input request.CreateFarm) (response.Farm, error) {
	f.calls++
	f.received = input
	f.ctx = ctx
	if f.err != nil {
		return response.Farm{}, f.err
	}
	return f.result, nil
}

type fakeListFarmsPort struct {
	result response.FarmList
	err    error

	calls int
	ctx   context.Context
}

func (f *fakeListFarmsPort) Execute(ctx context.Context, _ request.ListFarms) (response.FarmList, error) {
	f.calls++
	f.ctx = ctx
	if f.err != nil {
		return response.FarmList{Items: []response.Farm{}}, f.err
	}
	return f.result, nil
}

func newFarmMux(t *testing.T, create *fakeCreateFarmPort, list *fakeListFarmsPort, middleware controllers.Middleware) *http.ServeMux {
	t.Helper()

	mux := http.NewServeMux()
	controllers.NewFarmController(create, list, newResponder(t), discardLogger()).Register(mux, middleware)
	return mux
}

func postFarm(mux http.Handler, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/farms", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func getFarms(mux http.Handler) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/farms", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func markFarmContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, "marked")))
	})
}

func rejectFarmRequests(http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})
}

func farmPortErrors() []struct {
	name     string
	err      error
	want     int
	wantCode string
} {
	invalidName := exception.New(exception.CodeFarmInvalidName).WithParams(map[string]any{"min": 3, "max": 100})

	return []struct {
		name     string
		err      error
		want     int
		wantCode string
	}{
		{"coded validation", invalidName, http.StatusBadRequest, string(exception.CodeFarmInvalidName)},
		{"not a producer", exception.New(exception.CodeFarmInvalidProducer), http.StatusBadRequest, string(exception.CodeFarmInvalidProducer)},
		{"actor not found", exception.New(exception.CodeActorNotFound), http.StatusNotFound, string(exception.CodeActorNotFound)},
		{"coded forbidden", exception.New(exception.CodeForbidden), http.StatusForbidden, string(exception.CodeForbidden)},
		{"internal", exception.New(exception.CodeInternal).WithTechnical("boom"), http.StatusInternalServerError, string(exception.CodeInternal)},
		{"plain error", errors.New("boom"), http.StatusInternalServerError, string(exception.CodeInternal)},
	}
}

func TestCreateFarmReturnsCreated(t *testing.T) {
	port := &fakeCreateFarmPort{result: response.Farm{ID: "farm-1", Name: "La Esperanza", Altitude: 1650}}

	rec := postFarm(newFarmMux(t, port, &fakeListFarmsPort{}, passThrough), validFarmBody)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status: got %d, want %d (%s)", rec.Code, http.StatusCreated, rec.Body.String())
	}
	if got := rec.Header().Get("Location"); got != "/farms/farm-1" {
		t.Errorf("Location: got %q, want /farms/farm-1", got)
	}

	var body response.Farm
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("body is not JSON: %v", err)
	}
	if body != port.result {
		t.Errorf("body: got %+v, want %+v", body, port.result)
	}

	want := request.CreateFarm{Name: "La Esperanza", Altitude: 1650}
	if port.received != want {
		t.Errorf("port received %+v, want %+v", port.received, want)
	}
}

func TestCreateFarmRunsTheMiddleware(t *testing.T) {
	port := &fakeCreateFarmPort{}

	postFarm(newFarmMux(t, port, &fakeListFarmsPort{}, markFarmContext), validFarmBody)

	if port.calls != 1 || port.ctx.Value(ctxKey{}) != "marked" {
		t.Error("the request context from the middleware must reach the port")
	}
}

func TestCreateFarmStopsWhenMiddlewareRejects(t *testing.T) {
	port := &fakeCreateFarmPort{}

	rec := postFarm(newFarmMux(t, port, &fakeListFarmsPort{}, rejectFarmRequests), validFarmBody)

	if rec.Code != http.StatusUnauthorized || port.calls != 0 {
		t.Errorf("got status %d and %d port calls, want 401 and 0", rec.Code, port.calls)
	}
}

func TestCreateFarmRejectsInvalidBody(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{"empty body", ""},
		{"malformed json", `{"name":`},
		{"not an object", `["La Esperanza"]`},
		{"unknown field", `{"name":"La Esperanza","producer_id":"x"}`},
		{"altitude as text", `{"name":"La Esperanza","altitude":"1650"}`},
		{"decimal altitude", `{"name":"La Esperanza","altitude":1650.5}`},
		{"body too large", `{"name":"` + strings.Repeat("a", 1<<20) + `"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			port := &fakeCreateFarmPort{}

			rec := postFarm(newFarmMux(t, port, &fakeListFarmsPort{}, passThrough), tt.body)

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

func TestCreateFarmMapsPortErrors(t *testing.T) {
	for _, tt := range farmPortErrors() {
		t.Run(tt.name, func(t *testing.T) {
			rec := postFarm(newFarmMux(t, &fakeCreateFarmPort{err: tt.err}, &fakeListFarmsPort{}, passThrough), validFarmBody)

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

func TestListFarmsReturnsTheItems(t *testing.T) {
	list := &fakeListFarmsPort{result: response.FarmList{Items: []response.Farm{
		{ID: "farm-1", Name: "La Esperanza", Altitude: 1650},
		{ID: "farm-2", Name: "El Mirador", Altitude: 1800},
	}}}

	rec := getFarms(newFarmMux(t, &fakeCreateFarmPort{}, list, passThrough))

	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d (%s)", rec.Code, http.StatusOK, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type: got %q, want application/json", got)
	}

	var body response.FarmList
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("body is not JSON: %v", err)
	}
	if len(body.Items) != 2 || body.Items[0] != list.result.Items[0] || body.Items[1] != list.result.Items[1] {
		t.Errorf("items: got %+v, want %+v", body.Items, list.result.Items)
	}
}

func TestListFarmsWithoutFarmsReturnsAnEmptyArray(t *testing.T) {
	list := &fakeListFarmsPort{result: response.FarmList{Items: []response.Farm{}}}

	rec := getFarms(newFarmMux(t, &fakeCreateFarmPort{}, list, passThrough))

	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d", rec.Code, http.StatusOK)
	}
	if got := strings.TrimSpace(rec.Body.String()); got != `{"items":[]}` {
		t.Errorf("body: got %s, want {\"items\":[]}", got)
	}
}

func TestListFarmsRunsTheMiddleware(t *testing.T) {
	list := &fakeListFarmsPort{result: response.FarmList{Items: []response.Farm{}}}

	getFarms(newFarmMux(t, &fakeCreateFarmPort{}, list, markFarmContext))

	if list.calls != 1 || list.ctx.Value(ctxKey{}) != "marked" {
		t.Error("the request context from the middleware must reach the port")
	}
}

func TestListFarmsStopsWhenMiddlewareRejects(t *testing.T) {
	list := &fakeListFarmsPort{}

	rec := getFarms(newFarmMux(t, &fakeCreateFarmPort{}, list, rejectFarmRequests))

	if rec.Code != http.StatusUnauthorized || list.calls != 0 {
		t.Errorf("got status %d and %d port calls, want 401 and 0", rec.Code, list.calls)
	}
}

func TestListFarmsMapsPortErrors(t *testing.T) {
	for _, tt := range farmPortErrors() {
		t.Run(tt.name, func(t *testing.T) {
			rec := getFarms(newFarmMux(t, &fakeCreateFarmPort{}, &fakeListFarmsPort{err: tt.err}, passThrough))

			if rec.Code != tt.want {
				t.Errorf("status: got %d, want %d", rec.Code, tt.want)
			}

			body := decodeBody(t, rec)
			if body["code"] != tt.wantCode {
				t.Errorf("code: got %q, want %q", body["code"], tt.wantCode)
			}
		})
	}
}

func TestFarmsCollectionRejectsUnsupportedMethods(t *testing.T) {
	for _, method := range []string{http.MethodPut, http.MethodPatch, http.MethodDelete} {
		t.Run(method, func(t *testing.T) {
			create, list := &fakeCreateFarmPort{}, &fakeListFarmsPort{}
			req := httptest.NewRequest(method, "/farms", nil)
			rec := httptest.NewRecorder()

			newFarmMux(t, create, list, passThrough).ServeHTTP(rec, req)

			if rec.Code != http.StatusMethodNotAllowed || create.calls != 0 || list.calls != 0 {
				t.Errorf("got status %d, want 405 and no port calls", rec.Code)
			}
		})
	}
}
