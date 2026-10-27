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

const validRegisterBody = `{"type":"producer","name":"Finca La Esperanza SAS"}`

type fakeRegisterActorPort struct {
	result response.ActorRegistration
	err    error

	calls    int
	received request.RegisterActor
	ctx      context.Context
}

func (f *fakeRegisterActorPort) Execute(ctx context.Context, input request.RegisterActor) (response.ActorRegistration, error) {
	f.calls++
	f.received = input
	f.ctx = ctx
	if f.err != nil {
		return response.ActorRegistration{}, f.err
	}
	return f.result, nil
}

type fakeGetMePort struct {
	result response.Me
	err    error

	calls int
	ctx   context.Context
}

func (f *fakeGetMePort) Execute(ctx context.Context, _ request.GetMe) (response.Me, error) {
	f.calls++
	f.ctx = ctx
	if f.err != nil {
		return response.Me{}, f.err
	}
	return f.result, nil
}

func newMeMux(t *testing.T, register *fakeRegisterActorPort, me *fakeGetMePort, middleware controllers.Middleware) *http.ServeMux {
	t.Helper()

	mux := http.NewServeMux()
	controllers.NewMeController(register, me, newResponder(t), discardLogger()).Register(mux, middleware)
	return mux
}

func postMe(mux http.Handler, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/me", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func getMe(mux http.Handler) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func markMeContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, "marked")))
	})
}

func rejectMeRequests(http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})
}

func sampleActorResponse() response.Actor {
	return response.Actor{
		ID:        "actor-1",
		Type:      "producer",
		Name:      "Finca La Esperanza SAS",
		Email:     "producer@example.com",
		Status:    "active",
		CreatedAt: "2026-09-20T14:30:00Z",
	}
}

func sampleMeResponse() response.Me {
	return response.Me{Actor: sampleActorResponse(), FirstName: "Juan", LastName: "Ramírez"}
}

func mePortErrors() []struct {
	name     string
	err      error
	want     int
	wantCode string
} {
	return []struct {
		name     string
		err      error
		want     int
		wantCode string
	}{
		{"invalid type", exception.New(exception.CodeActorInvalidType), http.StatusBadRequest, string(exception.CodeActorInvalidType)},
		{"invalid name", exception.New(exception.CodeActorInvalidName).WithParams(map[string]any{"min": 3, "max": 100}), http.StatusBadRequest, string(exception.CodeActorInvalidName)},
		{"invalid email", exception.New(exception.CodeActorInvalidEmail), http.StatusBadRequest, string(exception.CodeActorInvalidEmail)},
		{"unauthorized", exception.New(exception.CodeUnauthorized), http.StatusUnauthorized, string(exception.CodeUnauthorized)},
		{"actor not found", exception.New(exception.CodeActorNotFound), http.StatusNotFound, string(exception.CodeActorNotFound)},
		{"forbidden", exception.New(exception.CodeForbidden), http.StatusForbidden, string(exception.CodeForbidden)},
		{"already exists", exception.New(exception.CodeActorAlreadyExists), http.StatusConflict, string(exception.CodeActorAlreadyExists)},
		{"internal", exception.New(exception.CodeInternal).WithTechnical("boom"), http.StatusInternalServerError, string(exception.CodeInternal)},
		{"plain error", errors.New("boom"), http.StatusInternalServerError, string(exception.CodeInternal)},
	}
}

func TestRegisterMeReturnsCreated(t *testing.T) {
	port := &fakeRegisterActorPort{result: response.ActorRegistration{Actor: sampleActorResponse(), Created: true}}

	rec := postMe(newMeMux(t, port, &fakeGetMePort{}, passThrough), validRegisterBody)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status: got %d, want %d (%s)", rec.Code, http.StatusCreated, rec.Body.String())
	}
	if got := rec.Header().Get("Location"); got != "/me" {
		t.Errorf("Location: got %q, want /me", got)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type: got %q, want application/json", got)
	}

	var body response.Actor
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("body is not JSON: %v", err)
	}
	if body != port.result.Actor {
		t.Errorf("body: got %+v, want %+v", body, port.result.Actor)
	}

	want := request.RegisterActor{Type: "producer", Name: "Finca La Esperanza SAS"}
	if port.received != want {
		t.Errorf("port received %+v, want %+v", port.received, want)
	}
}

func TestRegisterMeReturnsOKWhenAlreadyRegistered(t *testing.T) {
	port := &fakeRegisterActorPort{result: response.ActorRegistration{Actor: sampleActorResponse()}}

	rec := postMe(newMeMux(t, port, &fakeGetMePort{}, passThrough), validRegisterBody)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d (%s)", rec.Code, http.StatusOK, rec.Body.String())
	}
	if got := rec.Header().Get("Location"); got != "" {
		t.Errorf("Location: got %q, want none", got)
	}

	var body response.Actor
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("body is not JSON: %v", err)
	}
	if body != port.result.Actor {
		t.Errorf("body: got %+v, want %+v", body, port.result.Actor)
	}
}

func TestRegisterMeRunsTheMiddleware(t *testing.T) {
	port := &fakeRegisterActorPort{}

	postMe(newMeMux(t, port, &fakeGetMePort{}, markMeContext), validRegisterBody)

	if port.calls != 1 || port.ctx.Value(ctxKey{}) != "marked" {
		t.Error("the request context from the middleware must reach the port")
	}
}

func TestRegisterMeStopsWhenMiddlewareRejects(t *testing.T) {
	port := &fakeRegisterActorPort{}

	rec := postMe(newMeMux(t, port, &fakeGetMePort{}, rejectMeRequests), validRegisterBody)

	if rec.Code != http.StatusUnauthorized || port.calls != 0 {
		t.Errorf("got status %d and %d port calls, want 401 and 0", rec.Code, port.calls)
	}
}

func TestRegisterMeRejectsInvalidBody(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{"empty body", ""},
		{"malformed json", `{"type":`},
		{"not an object", `["producer"]`},
		{"unknown field", `{"type":"producer","name":"Finca La Esperanza SAS","email":"x@example.com"}`},
		{"user id in body", `{"type":"producer","name":"Finca La Esperanza SAS","user_id":"user_02"}`},
		{"type as number", `{"type":1,"name":"Finca La Esperanza SAS"}`},
		{"body too large", `{"name":"` + strings.Repeat("a", 1<<20) + `"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			port := &fakeRegisterActorPort{}

			rec := postMe(newMeMux(t, port, &fakeGetMePort{}, passThrough), tt.body)

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

func TestRegisterMeMapsPortErrors(t *testing.T) {
	for _, tt := range mePortErrors() {
		t.Run(tt.name, func(t *testing.T) {
			port := &fakeRegisterActorPort{err: tt.err}

			rec := postMe(newMeMux(t, port, &fakeGetMePort{}, passThrough), validRegisterBody)

			if rec.Code != tt.want {
				t.Errorf("status: got %d, want %d", rec.Code, tt.want)
			}
			body := decodeBody(t, rec)
			if body["code"] != tt.wantCode {
				t.Errorf("code: got %q, want %q", body["code"], tt.wantCode)
			}
			if body["error"] == "" {
				t.Error("error message must not be empty")
			}
			if rec.Header().Get("Location") != "" {
				t.Error("a failed registration must not set Location")
			}
		})
	}
}

func TestGetMeReturnsTheActor(t *testing.T) {
	port := &fakeGetMePort{result: sampleMeResponse()}

	rec := getMe(newMeMux(t, &fakeRegisterActorPort{}, port, passThrough))

	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d (%s)", rec.Code, http.StatusOK, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type: got %q, want application/json", got)
	}

	var body response.Me
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("body is not JSON: %v", err)
	}
	if body != port.result {
		t.Errorf("body: got %+v, want %+v", body, port.result)
	}
	if port.calls != 1 {
		t.Errorf("port calls: got %d, want 1", port.calls)
	}
}

func TestGetMeRunsTheMiddleware(t *testing.T) {
	port := &fakeGetMePort{}

	getMe(newMeMux(t, &fakeRegisterActorPort{}, port, markMeContext))

	if port.calls != 1 || port.ctx.Value(ctxKey{}) != "marked" {
		t.Error("the request context from the middleware must reach the port")
	}
}

func TestGetMeStopsWhenMiddlewareRejects(t *testing.T) {
	port := &fakeGetMePort{}

	rec := getMe(newMeMux(t, &fakeRegisterActorPort{}, port, rejectMeRequests))

	if rec.Code != http.StatusUnauthorized || port.calls != 0 {
		t.Errorf("got status %d and %d port calls, want 401 and 0", rec.Code, port.calls)
	}
}

func TestGetMeMapsPortErrors(t *testing.T) {
	for _, tt := range mePortErrors() {
		t.Run(tt.name, func(t *testing.T) {
			port := &fakeGetMePort{err: tt.err}

			rec := getMe(newMeMux(t, &fakeRegisterActorPort{}, port, passThrough))

			if rec.Code != tt.want {
				t.Errorf("status: got %d, want %d", rec.Code, tt.want)
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

func TestMeRoutesOnlyAcceptTheirMethods(t *testing.T) {
	mux := newMeMux(t, &fakeRegisterActorPort{}, &fakeGetMePort{}, passThrough)

	for _, method := range []string{http.MethodPut, http.MethodPatch, http.MethodDelete} {
		t.Run(method, func(t *testing.T) {
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, httptest.NewRequest(method, "/me", nil))

			if rec.Code != http.StatusMethodNotAllowed {
				t.Errorf("status: got %d, want %d", rec.Code, http.StatusMethodNotAllowed)
			}
		})
	}
}

func TestGetMeBodyIsFlatAndIncludesTheNames(t *testing.T) {
	port := &fakeGetMePort{result: sampleMeResponse()}

	rec := getMe(newMeMux(t, &fakeRegisterActorPort{}, port, passThrough))

	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("body is not JSON: %v", err)
	}
	want := map[string]any{
		"id":         "actor-1",
		"type":       "producer",
		"name":       "Finca La Esperanza SAS",
		"email":      "producer@example.com",
		"status":     "active",
		"created_at": "2026-09-20T14:30:00Z",
		"first_name": "Juan",
		"last_name":  "Ramírez",
	}
	if len(body) != len(want) {
		t.Errorf("body has %d fields, want %d: %v", len(body), len(want), body)
	}
	for key, value := range want {
		if body[key] != value {
			t.Errorf("%s: got %v, want %v", key, body[key], value)
		}
	}
}

func TestGetMeReturnsEmptyNamesWhenThePortHasNone(t *testing.T) {
	port := &fakeGetMePort{result: response.Me{Actor: sampleActorResponse()}}

	rec := getMe(newMeMux(t, &fakeRegisterActorPort{}, port, passThrough))

	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("body is not JSON: %v", err)
	}
	if body["first_name"] != "" || body["last_name"] != "" {
		t.Errorf("names: got (%v, %v), want empty strings", body["first_name"], body["last_name"])
	}
}

func TestRegisterMeDoesNotReturnNames(t *testing.T) {
	port := &fakeRegisterActorPort{result: response.ActorRegistration{Actor: sampleActorResponse(), Created: true}}

	rec := postMe(newMeMux(t, port, &fakeGetMePort{}, passThrough), validRegisterBody)

	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("body is not JSON: %v", err)
	}
	if _, ok := body["first_name"]; ok {
		t.Error("registering an actor must not return first_name")
	}
	if _, ok := body["last_name"]; ok {
		t.Error("registering an actor must not return last_name")
	}
}
