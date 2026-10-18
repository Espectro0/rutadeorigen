package controllers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"rutadeorigen/traceability/internal/application/input/dto/request"
	"rutadeorigen/traceability/internal/application/input/dto/response"
	"rutadeorigen/traceability/internal/infrastructure/input/controllers"
	"rutadeorigen/traceability/internal/platform/exception"
)

func newReadMux(t *testing.T, get *fakeGetLotPort, list *fakeListLotsPort, middleware controllers.Middleware) *http.ServeMux {
	t.Helper()

	mux := http.NewServeMux()
	controllers.NewLotController(&fakeCreateLotPort{}, get, list, newResponder(t), discardLogger()).Register(mux, middleware)
	return mux
}

func doGet(mux http.Handler, target string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, target, nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func TestGetLot(t *testing.T) {
	t.Run("returns the lot with its version as ETag", func(t *testing.T) {
		get := &fakeGetLotPort{result: response.Lot{ID: "lot-1", Code: "LOT-001", Status: "active", Version: 3}}

		rec := doGet(newReadMux(t, get, &fakeListLotsPort{}, passThrough), "/lots/lot-1")

		if rec.Code != http.StatusOK {
			t.Fatalf("status: got %d, want %d (%s)", rec.Code, http.StatusOK, rec.Body.String())
		}
		if got := rec.Header().Get("ETag"); got != `"3"` {
			t.Errorf("ETag: got %q, want \"3\"", got)
		}
		if got := rec.Header().Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type: got %q", got)
		}

		var body response.Lot
		if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
			t.Fatalf("body is not JSON: %v", err)
		}
		if body != get.result {
			t.Errorf("body: got %+v, want %+v", body, get.result)
		}
		if get.received.ID != "lot-1" {
			t.Errorf("port received %+v, want ID lot-1", get.received)
		}
	})

	t.Run("maps a missing lot to 404", func(t *testing.T) {
		get := &fakeGetLotPort{err: exception.New(exception.CodeLotNotFound)}

		rec := doGet(newReadMux(t, get, &fakeListLotsPort{}, passThrough), "/lots/lot-1")

		if rec.Code != http.StatusNotFound {
			t.Fatalf("status: got %d, want %d", rec.Code, http.StatusNotFound)
		}
		body := decodeBody(t, rec)
		if body["code"] != string(exception.CodeLotNotFound) || body["error"] == "" {
			t.Errorf("body: got %+v", body)
		}
		if rec.Header().Get("ETag") != "" {
			t.Error("an error response must not carry an ETag")
		}
	})

	t.Run("stops when the middleware rejects", func(t *testing.T) {
		get := &fakeGetLotPort{}
		reject := func(http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
			})
		}

		rec := doGet(newReadMux(t, get, &fakeListLotsPort{}, reject), "/lots/lot-1")

		if rec.Code != http.StatusUnauthorized || get.calls != 0 {
			t.Errorf("got status %d and %d port calls, want 401 and 0", rec.Code, get.calls)
		}
	})
}

func TestListLots(t *testing.T) {
	t.Run("returns the page and forwards the query parameters", func(t *testing.T) {
		list := &fakeListLotsPort{result: response.LotPage{
			Items:      []response.Lot{{ID: "lot-1"}, {ID: "lot-2"}},
			NextCursor: "next",
		}}

		rec := doGet(newReadMux(t, &fakeGetLotPort{}, list, passThrough),
			"/lots?farm_id=farm-1&custodian_id=actor-1&cursor=abc&limit=5")

		if rec.Code != http.StatusOK {
			t.Fatalf("status: got %d, want %d (%s)", rec.Code, http.StatusOK, rec.Body.String())
		}

		var body response.LotPage
		if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
			t.Fatalf("body is not JSON: %v", err)
		}
		if len(body.Items) != 2 || body.NextCursor != "next" {
			t.Errorf("body: got %+v", body)
		}

		want := request.ListLots{FarmID: "farm-1", CustodianID: "actor-1", Cursor: "abc", Limit: "5"}
		if list.received != want {
			t.Errorf("port received %+v, want %+v", list.received, want)
		}
	})

	t.Run("sends empty parameters when none are given", func(t *testing.T) {
		list := &fakeListLotsPort{}

		rec := doGet(newReadMux(t, &fakeGetLotPort{}, list, passThrough), "/lots")

		if rec.Code != http.StatusOK || list.received != (request.ListLots{}) {
			t.Errorf("got status %d and %+v", rec.Code, list.received)
		}
	})

	t.Run("maps validation errors to 400", func(t *testing.T) {
		list := &fakeListLotsPort{err: exception.New(exception.CodeInvalidCursor)}

		rec := doGet(newReadMux(t, &fakeGetLotPort{}, list, passThrough), "/lots?cursor=bad")

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status: got %d, want %d", rec.Code, http.StatusBadRequest)
		}
		body := decodeBody(t, rec)
		if body["code"] != string(exception.CodeInvalidCursor) || body["error"] == "" {
			t.Errorf("body: got %+v", body)
		}
	})

	t.Run("stops when the middleware rejects", func(t *testing.T) {
		list := &fakeListLotsPort{}
		reject := func(http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
			})
		}

		rec := doGet(newReadMux(t, &fakeGetLotPort{}, list, reject), "/lots")

		if rec.Code != http.StatusUnauthorized || list.calls != 0 {
			t.Errorf("got status %d and %d port calls, want 401 and 0", rec.Code, list.calls)
		}
	})
}
