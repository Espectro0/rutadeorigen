package main

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
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

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("body is not JSON: %v", err)
	}
	if _, err := uuid.Parse(body["id"]); err != nil {
		t.Errorf("id must be a UUID, got %q", body["id"])
	}
	if rec.Header().Get("Location") != "/lots/"+body["id"] {
		t.Errorf("Location: got %q", rec.Header().Get("Location"))
	}
	if body["custodian_id"] != producerHeader {
		t.Errorf("custodian must be the farm producer, got %q", body["custodian_id"])
	}
	if body["stage"] != "none" || body["status"] != "active" {
		t.Errorf("stage/status: got %q/%q, want none/active", body["stage"], body["status"])
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
