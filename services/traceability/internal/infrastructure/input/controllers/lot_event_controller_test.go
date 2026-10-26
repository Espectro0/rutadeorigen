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

const stageBody = `{"stage":"farm","event_date":"2026-09-20","note":"Dried on raised beds","details":{"drying_method":"sun_patio","drying_days":15}}`

type fakeRegisterStagePort struct {
	result response.RegisteredStage
	err    error

	calls    int
	received request.RegisterStage
	ctx      context.Context
}

func (f *fakeRegisterStagePort) Execute(ctx context.Context, input request.RegisterStage) (response.RegisteredStage, error) {
	f.calls++
	f.received = input
	f.ctx = ctx
	if f.err != nil {
		return response.RegisteredStage{}, f.err
	}
	return f.result, nil
}

func newEventMux(t *testing.T, port *fakeRegisterStagePort, middleware controllers.Middleware) *http.ServeMux {
	t.Helper()

	mux := http.NewServeMux()
	controllers.NewLotEventController(port, newResponder(t), discardLogger()).Register(mux, middleware)
	return mux
}

func postStage(mux http.Handler, lotID, ifMatch, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/lots/"+lotID+"/events", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if ifMatch != "" {
		req.Header.Set("If-Match", ifMatch)
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func registeredStage() response.RegisteredStage {
	return response.RegisteredStage{
		Event: response.LotEvent{
			ID:        "event-1",
			LotID:     "lot-1",
			Sequence:  1,
			Type:      "stage",
			Stage:     "farm",
			EventDate: "2026-09-20",
			Details:   json.RawMessage(`{"drying_days":15}`),
			Hash:      "abc",
		},
		LotVersion: 2,
	}
}

func TestRegisterStageReturnsCreated(t *testing.T) {
	port := &fakeRegisterStagePort{result: registeredStage()}

	rec := postStage(newEventMux(t, port, passThrough), "lot-1", `"1"`, stageBody)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status: got %d, want %d (%s)", rec.Code, http.StatusCreated, rec.Body.String())
	}
	if got := rec.Header().Get("ETag"); got != `"2"` {
		t.Errorf("ETag: got %q, want \"2\"", got)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type: got %q", got)
	}

	var body response.RegisteredStage
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("body is not JSON: %v", err)
	}
	if body.LotVersion != 2 || body.Event.ID != "event-1" || body.Event.Stage != "farm" || body.Event.Sequence != 1 {
		t.Errorf("unexpected body: %+v", body)
	}
	if string(body.Event.Details) != `{"drying_days":15}` {
		t.Errorf("Details: got %s", body.Event.Details)
	}
}

func TestRegisterStagePassesTheRequestToThePort(t *testing.T) {
	port := &fakeRegisterStagePort{result: registeredStage()}

	postStage(newEventMux(t, port, passThrough), "lot-1", `W/"7"`, stageBody)

	got := port.received
	if got.LotID != "lot-1" || got.IfMatch != `W/"7"` {
		t.Errorf("path and header: got lot %q, if-match %q", got.LotID, got.IfMatch)
	}
	if got.Stage != "farm" || got.EventDate != "2026-09-20" || got.Note != "Dried on raised beds" {
		t.Errorf("body fields: got %+v", got)
	}
	if string(got.Details) != `{"drying_method":"sun_patio","drying_days":15}` {
		t.Errorf("Details: got %s", got.Details)
	}
}

func TestRegisterStageIgnoresPathAndHeaderValuesInTheBody(t *testing.T) {
	port := &fakeRegisterStagePort{result: registeredStage()}

	rec := postStage(newEventMux(t, port, passThrough), "lot-1", `"1"`, `{"lot_id":"other","if_match":"9","stage":"farm"}`)

	if rec.Code != http.StatusBadRequest || port.calls != 0 {
		t.Errorf("got status %d and %d port calls, want 400 and 0", rec.Code, port.calls)
	}
}

func TestRegisterStageForwardsAMissingIfMatchToThePort(t *testing.T) {
	port := &fakeRegisterStagePort{err: exception.New(exception.CodeLotInvalidIfMatch)}

	rec := postStage(newEventMux(t, port, passThrough), "lot-1", "", stageBody)

	if port.calls != 1 || port.received.IfMatch != "" {
		t.Errorf("expected one call with an empty If-Match, got %d calls (%q)", port.calls, port.received.IfMatch)
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestRegisterStageRunsTheMiddleware(t *testing.T) {
	port := &fakeRegisterStagePort{result: registeredStage()}
	markContext := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, "marked")))
		})
	}

	postStage(newEventMux(t, port, markContext), "lot-1", `"1"`, stageBody)

	if port.calls != 1 || port.ctx.Value(ctxKey{}) != "marked" {
		t.Error("the request context from the middleware must reach the port")
	}
}

func TestRegisterStageStopsWhenMiddlewareRejects(t *testing.T) {
	port := &fakeRegisterStagePort{}
	reject := func(http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		})
	}

	rec := postStage(newEventMux(t, port, reject), "lot-1", `"1"`, stageBody)

	if rec.Code != http.StatusUnauthorized || port.calls != 0 {
		t.Errorf("got status %d and %d port calls, want 401 and 0", rec.Code, port.calls)
	}
}

func TestRegisterStageRejectsInvalidBody(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{"empty body", ""},
		{"malformed json", `{"stage":`},
		{"not an object", `["farm"]`},
		{"unknown field", `{"stage":"farm","stages":"x"}`},
		{"wrong field type", `{"stage":123}`},
		{"body too large", `{"note":"` + strings.Repeat("a", 1<<20) + `"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			port := &fakeRegisterStagePort{}

			rec := postStage(newEventMux(t, port, passThrough), "lot-1", `"1"`, tt.body)

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

func TestRegisterStageMapsPortErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   exception.Code
	}{
		{"lot not found", exception.New(exception.CodeLotNotFound), http.StatusNotFound, exception.CodeLotNotFound},
		{"invalid stage", exception.New(exception.CodeLotInvalidStage), http.StatusBadRequest, exception.CodeLotInvalidStage},
		{"invalid details", exception.New(exception.CodeStageInvalidDetails), http.StatusBadRequest, exception.CodeStageInvalidDetails},
		{"invalid event date", exception.New(exception.CodeLotInvalidEventDate), http.StatusBadRequest, exception.CodeLotInvalidEventDate},
		{"invalid if-match", exception.New(exception.CodeLotInvalidIfMatch), http.StatusBadRequest, exception.CodeLotInvalidIfMatch},
		{"version conflict", exception.New(exception.CodeLotVersionConflict), http.StatusConflict, exception.CodeLotVersionConflict},
		{"forbidden", exception.New(exception.CodeForbidden), http.StatusForbidden, exception.CodeForbidden},
		{"unexpected failure", context.DeadlineExceeded, http.StatusInternalServerError, exception.CodeInternal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			port := &fakeRegisterStagePort{err: tt.err}

			rec := postStage(newEventMux(t, port, passThrough), "lot-1", `"1"`, stageBody)

			if rec.Code != tt.wantStatus {
				t.Errorf("status: got %d, want %d", rec.Code, tt.wantStatus)
			}
			if got := decodeBody(t, rec)["code"]; got != string(tt.wantCode) {
				t.Errorf("code: got %q, want %q", got, tt.wantCode)
			}
			if rec.Header().Get("ETag") != "" {
				t.Error("a failed request must not set an ETag")
			}
		})
	}
}
