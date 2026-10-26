package main

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/google/uuid"

	"rutadeorigen/traceability/internal/platform/catalog"
	"rutadeorigen/traceability/internal/platform/exception"
)

const producerHeader = "11111111-1111-1111-1111-111111111111"

func newTestServer(t *testing.T) (*http.ServeMux, *bytes.Buffer) {
	t.Helper()

	messages, err := catalog.New()
	if err != nil {
		t.Fatalf("catalog.New: %v", err)
	}

	var logs bytes.Buffer
	mux := http.NewServeMux()
	registerRoutes(mux, slog.New(slog.NewJSONHandler(&logs, nil)), nil, messages)
	return mux, &logs
}

func createLot(mux http.Handler, actorID, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/lots", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if actorID != "" {
		req.Header.Set("X-Actor-ID", actorID)
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func lotBody(farmID, code, process, harvestDate string) string {
	return `{"farm_id":"` + farmID + `","code":"` + code + `","variety":"Castillo","process":"` +
		process + `","harvest_date":"` + harvestDate + `"}`
}

func TestCreateLotEndToEnd(t *testing.T) {
	mux, logs := newTestServer(t)

	rec := createLot(mux, producerHeader, lotBody(seedFarmID.String(), "LOT-001", "washed", "2026-01-15"))

	if rec.Code != http.StatusCreated {
		t.Fatalf("status: got %d, want %d (%s)", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var body struct {
		ID          string `json:"id"`
		CustodianID string `json:"custodian_id"`
		Stage       string `json:"stage"`
		Status      string `json:"status"`
		Version     int    `json:"version"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("body is not JSON: %v", err)
	}
	if _, err := uuid.Parse(body.ID); err != nil {
		t.Errorf("id must be a UUID, got %q", body.ID)
	}
	if rec.Header().Get("Location") != "/lots/"+body.ID {
		t.Errorf("Location: got %q", rec.Header().Get("Location"))
	}
	if body.CustodianID != producerHeader {
		t.Errorf("custodian must be the farm producer, got %q", body.CustodianID)
	}
	if body.Stage != "none" || body.Status != "active" {
		t.Errorf("stage/status: got %q/%q, want none/active", body.Stage, body.Status)
	}
	if body.Version != 1 {
		t.Errorf("version: got %d, want 1", body.Version)
	}
	if !strings.Contains(logs.String(), `"event_type":"lot.created"`) {
		t.Errorf("lot.created event was not published, logs: %s", logs.String())
	}
}

func TestCreateLotEndToEndErrors(t *testing.T) {
	farm := seedFarmID.String()

	tests := []struct {
		name     string
		actorID  string
		body     string
		want     int
		wantCode string
	}{
		{"missing actor", "", lotBody(farm, "LOT-001", "washed", "2026-01-15"), http.StatusUnauthorized, "UNAUTHORIZED"},
		{"unknown actor", uuid.NewString(), lotBody(farm, "LOT-001", "washed", "2026-01-15"), http.StatusNotFound, string(exception.CodeActorNotFound)},
		{"malformed farm id", producerHeader, lotBody("abc", "LOT-001", "washed", "2026-01-15"), http.StatusBadRequest, string(exception.CodeLotInvalidFarmID)},
		{"unknown farm", producerHeader, lotBody(uuid.NewString(), "LOT-001", "washed", "2026-01-15"), http.StatusNotFound, string(exception.CodeFarmNotFound)},
		{"unknown process", producerHeader, lotBody(farm, "LOT-001", "fermented", "2026-01-15"), http.StatusBadRequest, string(exception.CodeLotInvalidProcess)},
		{"future harvest date", producerHeader, lotBody(farm, "LOT-001", "washed", "2999-01-01"), http.StatusBadRequest, string(exception.CodeLotInvalidHarvestDate)},
		{"code too short", producerHeader, lotBody(farm, "L1", "washed", "2026-01-15"), http.StatusBadRequest, string(exception.CodeLotInvalidCode)},
		{"unknown json field", producerHeader, `{"farm_id":"` + farm + `","x":1}`, http.StatusBadRequest, "INVALID_REQUEST_BODY"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux, _ := newTestServer(t)

			rec := createLot(mux, tt.actorID, tt.body)

			if rec.Code != tt.want {
				t.Fatalf("status: got %d, want %d (%s)", rec.Code, tt.want, rec.Body.String())
			}

			var body map[string]string
			if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
				t.Fatalf("body is not JSON: %v", err)
			}
			if body["code"] != tt.wantCode {
				t.Errorf("code: got %q, want %q", body["code"], tt.wantCode)
			}
			if body["error"] == "" {
				t.Error("error message must not be empty")
			}
		})
	}
}

func TestCreateLotEndToEndDuplicatedCode(t *testing.T) {
	mux, _ := newTestServer(t)
	body := lotBody(seedFarmID.String(), "LOT-001", "washed", "2026-01-15")

	if rec := createLot(mux, producerHeader, body); rec.Code != http.StatusCreated {
		t.Fatalf("first request: got %d, want %d", rec.Code, http.StatusCreated)
	}
	rec := createLot(mux, producerHeader, body)

	if rec.Code != http.StatusConflict {
		t.Errorf("second request: got %d, want %d", rec.Code, http.StatusConflict)
	}
}

func TestSeedData(t *testing.T) {
	actors := seedActors()
	farms := seedFarms()

	if len(actors) != 1 || actors[0].ID() != seedProducerID || !actors[0].IsActive() || !actors[0].IsProducer() {
		t.Errorf("seed must contain one active producer with the fixed id, got %+v", actors)
	}
	if len(farms) != 1 || farms[0].ID() != seedFarmID || !farms[0].BelongsTo(seedProducerID) {
		t.Errorf("seed must contain one farm owned by the producer, got %+v", farms)
	}
}

func get(mux http.Handler, actorID, target string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, target, nil)
	if actorID != "" {
		req.Header.Set("X-Actor-ID", actorID)
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

type lotView struct {
	ID      string `json:"id"`
	Code    string `json:"code"`
	Version int    `json:"version"`
}

type pageView struct {
	Items      []lotView `json:"items"`
	NextCursor string    `json:"next_cursor"`
}

func mustCreateLot(t *testing.T, mux http.Handler, code string) lotView {
	t.Helper()

	rec := createLot(mux, producerHeader, lotBody(seedFarmID.String(), code, "washed", "2026-01-15"))
	if rec.Code != http.StatusCreated {
		t.Fatalf("create %s: status %d (%s)", code, rec.Code, rec.Body.String())
	}

	var created lotView
	if err := json.NewDecoder(rec.Body).Decode(&created); err != nil {
		t.Fatalf("body is not JSON: %v", err)
	}
	return created
}

func decodePage(t *testing.T, rec *httptest.ResponseRecorder) pageView {
	t.Helper()

	var page pageView
	if err := json.NewDecoder(rec.Body).Decode(&page); err != nil {
		t.Fatalf("body is not JSON: %v", err)
	}
	return page
}

func TestGetLotEndToEnd(t *testing.T) {
	mux, _ := newTestServer(t)
	created := mustCreateLot(t, mux, "LOT-001")

	t.Run("returns the created lot with its version as ETag", func(t *testing.T) {
		rec := get(mux, producerHeader, "/lots/"+created.ID)

		if rec.Code != http.StatusOK {
			t.Fatalf("status: got %d, want %d (%s)", rec.Code, http.StatusOK, rec.Body.String())
		}
		if got := rec.Header().Get("ETag"); got != `"1"` {
			t.Errorf("ETag: got %q, want \"1\"", got)
		}

		var body lotView
		if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
			t.Fatalf("body is not JSON: %v", err)
		}
		if body.ID != created.ID || body.Code != "LOT-001" || body.Version != 1 {
			t.Errorf("body: got %+v", body)
		}
	})

	tests := []struct {
		name     string
		actorID  string
		target   string
		want     int
		wantCode string
	}{
		{"unknown lot", producerHeader, "/lots/" + uuid.NewString(), http.StatusNotFound, string(exception.CodeLotNotFound)},
		{"malformed id", producerHeader, "/lots/not-a-uuid", http.StatusNotFound, string(exception.CodeLotNotFound)},
		{"missing actor", "", "/lots/" + created.ID, http.StatusUnauthorized, "UNAUTHORIZED"},
		{"unknown actor", uuid.NewString(), "/lots/" + created.ID, http.StatusNotFound, string(exception.CodeActorNotFound)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := get(mux, tt.actorID, tt.target)

			if rec.Code != tt.want {
				t.Fatalf("status: got %d, want %d (%s)", rec.Code, tt.want, rec.Body.String())
			}
			var body map[string]string
			if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
				t.Fatalf("body is not JSON: %v", err)
			}
			if body["code"] != tt.wantCode {
				t.Errorf("code: got %q, want %q", body["code"], tt.wantCode)
			}
		})
	}
}

func TestListLotsEndToEnd(t *testing.T) {
	mux, _ := newTestServer(t)
	first := mustCreateLot(t, mux, "LOT-001")
	second := mustCreateLot(t, mux, "LOT-002")

	t.Run("lists every lot without a next cursor", func(t *testing.T) {
		rec := get(mux, producerHeader, "/lots")

		if rec.Code != http.StatusOK {
			t.Fatalf("status: got %d, want %d (%s)", rec.Code, http.StatusOK, rec.Body.String())
		}
		page := decodePage(t, rec)
		if len(page.Items) != 2 || page.NextCursor != "" {
			t.Errorf("got %d items and cursor %q, want 2 and none", len(page.Items), page.NextCursor)
		}
	})

	t.Run("filters by farm and custodian", func(t *testing.T) {
		own := get(mux, producerHeader, "/lots?farm_id="+seedFarmID.String()+"&custodian_id="+producerHeader)
		other := get(mux, producerHeader, "/lots?farm_id="+uuid.NewString())

		if got := len(decodePage(t, own).Items); got != 2 {
			t.Errorf("own farm: got %d items, want 2", got)
		}
		if got := decodePage(t, other).Items; len(got) != 0 {
			t.Errorf("other farm: got %d items, want 0", len(got))
		}
	})

	t.Run("pages through the lots with the cursor", func(t *testing.T) {
		firstPage := decodePage(t, get(mux, producerHeader, "/lots?limit=1"))
		if len(firstPage.Items) != 1 || firstPage.NextCursor == "" {
			t.Fatalf("first page: got %d items and cursor %q", len(firstPage.Items), firstPage.NextCursor)
		}

		secondPage := decodePage(t, get(mux, producerHeader, "/lots?limit=1&cursor="+firstPage.NextCursor))
		if len(secondPage.Items) != 1 || secondPage.NextCursor != "" {
			t.Fatalf("second page: got %d items and cursor %q", len(secondPage.Items), secondPage.NextCursor)
		}

		seen := map[string]bool{firstPage.Items[0].ID: true, secondPage.Items[0].ID: true}
		if len(seen) != 2 || !seen[first.ID] || !seen[second.ID] {
			t.Errorf("pages returned %v, want both created lots exactly once", seen)
		}
	})

	tests := []struct {
		name     string
		actorID  string
		target   string
		want     int
		wantCode string
	}{
		{"invalid limit", producerHeader, "/lots?limit=0", http.StatusBadRequest, string(exception.CodeInvalidLimit)},
		{"invalid cursor", producerHeader, "/lots?cursor=***", http.StatusBadRequest, string(exception.CodeInvalidCursor)},
		{"invalid farm id", producerHeader, "/lots?farm_id=nope", http.StatusBadRequest, string(exception.CodeLotInvalidFarmID)},
		{"invalid custodian id", producerHeader, "/lots?custodian_id=nope", http.StatusBadRequest, string(exception.CodeLotInvalidCustodianID)},
		{"missing actor", "", "/lots", http.StatusUnauthorized, "UNAUTHORIZED"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := get(mux, tt.actorID, tt.target)

			if rec.Code != tt.want {
				t.Fatalf("status: got %d, want %d (%s)", rec.Code, tt.want, rec.Body.String())
			}
			var body map[string]string
			if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
				t.Fatalf("body is not JSON: %v", err)
			}
			if body["code"] != tt.wantCode {
				t.Errorf("code: got %q, want %q", body["code"], tt.wantCode)
			}
		})
	}
}

type farmView struct {
	ID         string `json:"id"`
	ProducerID string `json:"producer_id"`
	Name       string `json:"name"`
	Altitude   int    `json:"altitude"`
	CreatedAt  string `json:"created_at"`
}

type farmListView struct {
	Items []farmView `json:"items"`
}

func farmBody(name string, altitude int) string {
	return `{"name":"` + name + `","altitude":` + strconv.Itoa(altitude) + `}`
}

func createFarm(mux http.Handler, actorID, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/farms", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if actorID != "" {
		req.Header.Set("X-Actor-ID", actorID)
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func mustCreateFarm(t *testing.T, mux http.Handler, name string) farmView {
	t.Helper()

	rec := createFarm(mux, producerHeader, farmBody(name, 1650))
	if rec.Code != http.StatusCreated {
		t.Fatalf("create farm %s: status %d (%s)", name, rec.Code, rec.Body.String())
	}

	var created farmView
	if err := json.NewDecoder(rec.Body).Decode(&created); err != nil {
		t.Fatalf("body is not JSON: %v", err)
	}
	return created
}

func decodeFarms(t *testing.T, rec *httptest.ResponseRecorder) farmListView {
	t.Helper()

	var list farmListView
	if err := json.NewDecoder(rec.Body).Decode(&list); err != nil {
		t.Fatalf("body is not JSON: %v", err)
	}
	return list
}

func assertErrorCode(t *testing.T, rec *httptest.ResponseRecorder, want int, wantCode string) {
	t.Helper()

	if rec.Code != want {
		t.Fatalf("status: got %d, want %d (%s)", rec.Code, want, rec.Body.String())
	}
	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("body is not JSON: %v", err)
	}
	if body["code"] != wantCode {
		t.Errorf("code: got %q, want %q", body["code"], wantCode)
	}
	if body["error"] == "" {
		t.Error("error message must not be empty")
	}
}

func TestCreateFarmEndToEnd(t *testing.T) {
	mux, _ := newTestServer(t)

	rec := createFarm(mux, producerHeader, farmBody("  La Esperanza  ", 1650))

	if rec.Code != http.StatusCreated {
		t.Fatalf("status: got %d, want %d (%s)", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var body farmView
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("body is not JSON: %v", err)
	}
	if _, err := uuid.Parse(body.ID); err != nil {
		t.Errorf("id must be a UUID, got %q", body.ID)
	}
	if rec.Header().Get("Location") != "/farms/"+body.ID {
		t.Errorf("Location: got %q", rec.Header().Get("Location"))
	}
	if body.ProducerID != producerHeader {
		t.Errorf("producer must be the actor of the request, got %q", body.ProducerID)
	}
	if body.Name != "La Esperanza" || body.Altitude != 1650 {
		t.Errorf("name/altitude: got %q/%d, want La Esperanza/1650", body.Name, body.Altitude)
	}
	if body.CreatedAt == "" {
		t.Error("created_at must not be empty")
	}
}

func TestCreateFarmEndToEndErrors(t *testing.T) {
	tests := []struct {
		name     string
		actorID  string
		body     string
		want     int
		wantCode string
	}{
		{"missing actor", "", farmBody("La Esperanza", 1650), http.StatusUnauthorized, "UNAUTHORIZED"},
		{"unknown actor", uuid.NewString(), farmBody("La Esperanza", 1650), http.StatusNotFound, string(exception.CodeActorNotFound)},
		{"name too short", producerHeader, farmBody("LE", 1650), http.StatusBadRequest, string(exception.CodeFarmInvalidName)},
		{"negative altitude", producerHeader, farmBody("La Esperanza", -1), http.StatusBadRequest, string(exception.CodeFarmInvalidAltitude)},
		{"altitude above the limit", producerHeader, farmBody("La Esperanza", 3001), http.StatusBadRequest, string(exception.CodeFarmInvalidAltitude)},
		{"altitude as text", producerHeader, `{"name":"La Esperanza","altitude":"1650"}`, http.StatusBadRequest, "INVALID_REQUEST_BODY"},
		{"unknown json field", producerHeader, `{"name":"La Esperanza","producer_id":"x"}`, http.StatusBadRequest, "INVALID_REQUEST_BODY"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux, _ := newTestServer(t)

			assertErrorCode(t, createFarm(mux, tt.actorID, tt.body), tt.want, tt.wantCode)
		})
	}
}

func TestListFarmsEndToEnd(t *testing.T) {
	mux, _ := newTestServer(t)

	t.Run("lists the seeded farm of the producer", func(t *testing.T) {
		rec := get(mux, producerHeader, "/farms")

		if rec.Code != http.StatusOK {
			t.Fatalf("status: got %d, want %d (%s)", rec.Code, http.StatusOK, rec.Body.String())
		}
		list := decodeFarms(t, rec)
		if len(list.Items) != 1 || list.Items[0].ID != seedFarmID.String() {
			t.Errorf("got %+v, want only the seeded farm", list.Items)
		}
	})

	created := mustCreateFarm(t, mux, "El Mirador")

	t.Run("includes the created farm first", func(t *testing.T) {
		list := decodeFarms(t, get(mux, producerHeader, "/farms"))

		if len(list.Items) != 2 {
			t.Fatalf("got %d farms, want 2", len(list.Items))
		}
		if list.Items[0].ID != created.ID || list.Items[1].ID != seedFarmID.String() {
			t.Errorf("got %+v, want the created farm and then the seeded one", list.Items)
		}
		for _, f := range list.Items {
			if f.ProducerID != producerHeader {
				t.Errorf("farm %s belongs to %s, want %s", f.ID, f.ProducerID, producerHeader)
			}
		}
	})

	t.Run("missing actor", func(t *testing.T) {
		assertErrorCode(t, get(mux, "", "/farms"), http.StatusUnauthorized, "UNAUTHORIZED")
	})

	t.Run("unknown actor", func(t *testing.T) {
		assertErrorCode(t, get(mux, uuid.NewString(), "/farms"), http.StatusNotFound, string(exception.CodeActorNotFound))
	})
}

func TestCreateLotOnACreatedFarmEndToEnd(t *testing.T) {
	mux, logs := newTestServer(t)
	created := mustCreateFarm(t, mux, "La Esperanza")

	rec := createLot(mux, producerHeader, lotBody(created.ID, "LOT-001", "washed", "2026-01-15"))

	if rec.Code != http.StatusCreated {
		t.Fatalf("status: got %d, want %d (%s)", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var lot struct {
		ID          string `json:"id"`
		FarmID      string `json:"farm_id"`
		CustodianID string `json:"custodian_id"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&lot); err != nil {
		t.Fatalf("body is not JSON: %v", err)
	}
	if lot.FarmID != created.ID || lot.CustodianID != producerHeader {
		t.Errorf("farm/custodian: got %q/%q, want %q/%q", lot.FarmID, lot.CustodianID, created.ID, producerHeader)
	}

	page := decodePage(t, get(mux, producerHeader, "/lots?farm_id="+created.ID))
	if len(page.Items) != 1 || page.Items[0].ID != lot.ID {
		t.Errorf("lots of the created farm: got %+v, want only the new lot", page.Items)
	}

	if !strings.Contains(logs.String(), `"name":"La Esperanza"`) {
		t.Errorf("lot.created must carry the created farm snapshot, logs: %s", logs.String())
	}
}

const (
	farmStageBody      = `{"stage":"farm","event_date":"2026-02-01","note":"Dried on raised beds","details":{"drying_method":"sun_patio","drying_days":15,"moisture_percent":11.5,"parchment_weight_kg":120.5}}`
	roastingStageBody  = `{"stage":"roasting","event_date":"2026-02-10","details":{"roast_level":"medium","duration_minutes":14,"green_weight_kg":100,"roasted_weight_kg":84}}`
	packagingStageBody = `{"stage":"packaging","event_date":"2026-02-20","details":{"grind":"whole_bean","package_weight_g":250,"units":40,"best_before":"2027-02-20"}}`
	genesisHash        = "0000000000000000000000000000000000000000000000000000000000000000"
)

type stageView struct {
	Event struct {
		ID           string          `json:"id"`
		LotID        string          `json:"lot_id"`
		Sequence     int             `json:"sequence"`
		Type         string          `json:"type"`
		Stage        string          `json:"stage"`
		EventDate    string          `json:"event_date"`
		Note         string          `json:"note"`
		ActorID      string          `json:"actor_id"`
		ActorRole    string          `json:"actor_role"`
		Details      json.RawMessage `json:"details"`
		PreviousHash string          `json:"previous_hash"`
		Hash         string          `json:"hash"`
		RegisteredAt string          `json:"registered_at"`
	} `json:"event"`
	LotVersion int `json:"lot_version"`
}

func postStage(mux http.Handler, actorID, lotID, ifMatch, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/lots/"+lotID+"/events", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if actorID != "" {
		req.Header.Set("X-Actor-ID", actorID)
	}
	if ifMatch != "" {
		req.Header.Set("If-Match", ifMatch)
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func mustRegisterStage(t *testing.T, mux http.Handler, lotID, ifMatch, body string) (stageView, string) {
	t.Helper()

	rec := postStage(mux, producerHeader, lotID, ifMatch, body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("register stage: status %d (%s)", rec.Code, rec.Body.String())
	}

	var view stageView
	if err := json.NewDecoder(rec.Body).Decode(&view); err != nil {
		t.Fatalf("body is not JSON: %v", err)
	}
	return view, rec.Header().Get("ETag")
}

func TestRegisterStageEndToEnd(t *testing.T) {
	mux, logs := newTestServer(t)
	created := mustCreateLot(t, mux, "LOT-001")

	farm, farmETag := mustRegisterStage(t, mux, created.ID, `"1"`, farmStageBody)
	roasting, roastingETag := mustRegisterStage(t, mux, created.ID, farmETag, roastingStageBody)
	packaging, packagingETag := mustRegisterStage(t, mux, created.ID, roastingETag, packagingStageBody)

	t.Run("each registration bumps the lot version and returns it as ETag", func(t *testing.T) {
		if farmETag != `"2"` || roastingETag != `"3"` || packagingETag != `"4"` {
			t.Errorf("ETags: got %s, %s, %s", farmETag, roastingETag, packagingETag)
		}
		if farm.LotVersion != 2 || roasting.LotVersion != 3 || packaging.LotVersion != 4 {
			t.Errorf("lot versions: got %d, %d, %d", farm.LotVersion, roasting.LotVersion, packaging.LotVersion)
		}
	})

	t.Run("the events are chained from the genesis hash", func(t *testing.T) {
		if farm.Event.Sequence != 1 || roasting.Event.Sequence != 2 || packaging.Event.Sequence != 3 {
			t.Errorf("sequences: got %d, %d, %d", farm.Event.Sequence, roasting.Event.Sequence, packaging.Event.Sequence)
		}
		if farm.Event.PreviousHash != genesisHash {
			t.Errorf("first previous hash: got %q", farm.Event.PreviousHash)
		}
		if roasting.Event.PreviousHash != farm.Event.Hash || packaging.Event.PreviousHash != roasting.Event.Hash {
			t.Error("each event must point to the hash of the previous one")
		}
		for _, e := range []string{farm.Event.Hash, roasting.Event.Hash, packaging.Event.Hash} {
			if len(e) != 64 {
				t.Errorf("hash must have 64 hex characters, got %q", e)
			}
		}
	})

	t.Run("the event carries the request data and the actor", func(t *testing.T) {
		if _, err := uuid.Parse(farm.Event.ID); err != nil {
			t.Errorf("id must be a UUID, got %q", farm.Event.ID)
		}
		if farm.Event.LotID != created.ID || farm.Event.Type != "stage" || farm.Event.Stage != "farm" {
			t.Errorf("unexpected event: %+v", farm.Event)
		}
		if farm.Event.EventDate != "2026-02-01" || farm.Event.Note != "Dried on raised beds" {
			t.Errorf("date/note: got %q/%q", farm.Event.EventDate, farm.Event.Note)
		}
		if farm.Event.ActorID != producerHeader || farm.Event.ActorRole != "producer" {
			t.Errorf("actor: got %q/%q", farm.Event.ActorID, farm.Event.ActorRole)
		}
		if farm.Event.RegisteredAt == "" {
			t.Error("registered_at must not be empty")
		}

		var details map[string]any
		if err := json.Unmarshal(farm.Event.Details, &details); err != nil {
			t.Fatalf("details are not JSON: %v", err)
		}
		if details["drying_method"] != "sun_patio" || details["drying_days"] != float64(15) {
			t.Errorf("details: got %v", details)
		}
	})

	t.Run("the lot reflects the last stage and version", func(t *testing.T) {
		rec := get(mux, producerHeader, "/lots/"+created.ID)

		if rec.Code != http.StatusOK {
			t.Fatalf("status: got %d (%s)", rec.Code, rec.Body.String())
		}
		if rec.Header().Get("ETag") != `"4"` {
			t.Errorf("ETag: got %q, want \"4\"", rec.Header().Get("ETag"))
		}
		var body struct {
			Stage   string `json:"stage"`
			Version int    `json:"version"`
		}
		if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
			t.Fatalf("body is not JSON: %v", err)
		}
		if body.Stage != "packaging" || body.Version != 4 {
			t.Errorf("got stage %q and version %d, want packaging and 4", body.Stage, body.Version)
		}
	})

	t.Run("a stage_registered event is published per registration", func(t *testing.T) {
		if got := strings.Count(logs.String(), `"event_type":"lot.stage_registered"`); got != 3 {
			t.Errorf("got %d lot.stage_registered events, want 3, logs: %s", got, logs.String())
		}
	})
}

func TestRegisterStageEndToEndAcceptsWeakAndUnquotedIfMatch(t *testing.T) {
	mux, _ := newTestServer(t)
	created := mustCreateLot(t, mux, "LOT-001")

	_, etag := mustRegisterStage(t, mux, created.ID, `W/"1"`, farmStageBody)
	_, _ = mustRegisterStage(t, mux, created.ID, strings.Trim(etag, `"`), roastingStageBody)
}

func TestRegisterStageEndToEndErrors(t *testing.T) {
	tests := []struct {
		name     string
		actorID  string
		lotID    func(created lotView) string
		ifMatch  string
		body     string
		want     int
		wantCode string
	}{
		{"missing actor", "", sameLot, `"1"`, farmStageBody, http.StatusUnauthorized, "UNAUTHORIZED"},
		{"unknown actor", uuid.NewString(), sameLot, `"1"`, farmStageBody, http.StatusNotFound, string(exception.CodeActorNotFound)},
		{"unknown lot", producerHeader, otherLot, `"1"`, farmStageBody, http.StatusNotFound, string(exception.CodeLotNotFound)},
		{"malformed lot id", producerHeader, malformedLot, `"1"`, farmStageBody, http.StatusNotFound, string(exception.CodeLotNotFound)},
		{"missing if-match", producerHeader, sameLot, "", farmStageBody, http.StatusBadRequest, string(exception.CodeLotInvalidIfMatch)},
		{"malformed if-match", producerHeader, sameLot, `"abc"`, farmStageBody, http.StatusBadRequest, string(exception.CodeLotInvalidIfMatch)},
		{"stale if-match", producerHeader, sameLot, `"2"`, farmStageBody, http.StatusConflict, string(exception.CodeLotVersionConflict)},
		{"skipped stage", producerHeader, sameLot, `"1"`, roastingStageBody, http.StatusConflict, string(exception.CodeLotStageOutOfOrder)},
		{"unknown stage", producerHeader, sameLot, `"1"`, `{"stage":"drying","event_date":"2026-02-01","details":{}}`, http.StatusBadRequest, string(exception.CodeLotInvalidStage)},
		{"stage none", producerHeader, sameLot, `"1"`, `{"stage":"none","event_date":"2026-02-01","details":{}}`, http.StatusBadRequest, string(exception.CodeLotInvalidStage)},
		{"invalid event date", producerHeader, sameLot, `"1"`, strings.Replace(farmStageBody, "2026-02-01", "01/02/2026", 1), http.StatusBadRequest, string(exception.CodeLotInvalidEventDate)},
		{"event date before harvest", producerHeader, sameLot, `"1"`, strings.Replace(farmStageBody, "2026-02-01", "2026-01-01", 1), http.StatusBadRequest, string(exception.CodeLotInvalidEventDate)},
		{"future event date", producerHeader, sameLot, `"1"`, strings.Replace(farmStageBody, "2026-02-01", "2999-01-01", 1), http.StatusBadRequest, string(exception.CodeLotInvalidEventDate)},
		{"missing details", producerHeader, sameLot, `"1"`, `{"stage":"farm","event_date":"2026-02-01"}`, http.StatusBadRequest, string(exception.CodeStageInvalidDetails)},
		{"unknown detail field", producerHeader, sameLot, `"1"`, strings.Replace(farmStageBody, `"drying_days":15`, `"drying_days":15,"altitude":1`, 1), http.StatusBadRequest, string(exception.CodeStageInvalidDetails)},
		{"details of another stage", producerHeader, sameLot, `"1"`, `{"stage":"farm","event_date":"2026-02-01","details":{"roast_level":"medium"}}`, http.StatusBadRequest, string(exception.CodeStageInvalidDetails)},
		{"moisture out of range", producerHeader, sameLot, `"1"`, strings.Replace(farmStageBody, "11.5", "30", 1), http.StatusBadRequest, string(exception.CodeStageInvalidMoisture)},
		{"unknown json field", producerHeader, sameLot, `"1"`, `{"stage":"farm","lot_id":"x"}`, http.StatusBadRequest, "INVALID_REQUEST_BODY"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux, logs := newTestServer(t)
			created := mustCreateLot(t, mux, "LOT-001")

			rec := postStage(mux, tt.actorID, tt.lotID(created), tt.ifMatch, tt.body)

			assertErrorCode(t, rec, tt.want, tt.wantCode)
			if rec.Header().Get("ETag") != "" {
				t.Error("a failed request must not set an ETag")
			}
			if strings.Contains(logs.String(), `"event_type":"lot.stage_registered"`) {
				t.Error("a failed request must not publish an event")
			}

			after := get(mux, producerHeader, "/lots/"+created.ID)
			if after.Header().Get("ETag") != `"1"` {
				t.Errorf("the lot must stay untouched, ETag %q", after.Header().Get("ETag"))
			}
		})
	}
}

func sameLot(created lotView) string { return created.ID }
func otherLot(lotView) string        { return uuid.NewString() }
func malformedLot(lotView) string    { return "not-a-uuid" }

func TestRegisterStageEndToEndRejectsAStaleVersionAfterAnotherRegistration(t *testing.T) {
	mux, _ := newTestServer(t)
	created := mustCreateLot(t, mux, "LOT-001")
	mustRegisterStage(t, mux, created.ID, `"1"`, farmStageBody)

	rec := postStage(mux, producerHeader, created.ID, `"1"`, roastingStageBody)

	assertErrorCode(t, rec, http.StatusConflict, string(exception.CodeLotVersionConflict))
}

func TestRegisterStageEndToEndDoesNotAdvanceTheSameStageTwice(t *testing.T) {
	mux, _ := newTestServer(t)
	created := mustCreateLot(t, mux, "LOT-001")
	_, etag := mustRegisterStage(t, mux, created.ID, `"1"`, farmStageBody)

	rec := postStage(mux, producerHeader, created.ID, etag, farmStageBody)

	assertErrorCode(t, rec, http.StatusConflict, string(exception.CodeLotStageOutOfOrder))
}
