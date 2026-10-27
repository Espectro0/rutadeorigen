package main

import (
	"bytes"
	"context"
	"encoding/json"
	"image/png"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	libmessaging "rutadeorigen/lib/messaging"
	"rutadeorigen/qr/internal/infrastructure/input/controllers"
	inmessaging "rutadeorigen/qr/internal/infrastructure/input/messaging"
	"rutadeorigen/qr/internal/platform/catalog"
)

type harness struct {
	mux        *http.ServeMux
	lotCreated *inmessaging.LotEventsHandler
}

func newHarness(t *testing.T) harness {
	t.Helper()

	return newHarnessWith(t, authn{mode: authModeDevHeader})
}

func newHarnessWith(t *testing.T, auth authn) harness {
	t.Helper()

	messages, err := catalog.New()
	if err != nil {
		t.Fatalf("catalog: %v", err)
	}

	mux := http.NewServeMux()
	handler := registerRoutes(mux, slog.New(slog.NewTextHandler(io.Discard, nil)), messages, dependencies{
		publicBaseURL: "https://rutadeorigen.co",
		cacheTTL:      time.Minute,
	}, auth)
	return harness{mux: mux, lotCreated: handler}
}

func (h harness) publish(t *testing.T, messageID uuid.UUID, lotID uuid.UUID, stage string) error {
	t.Helper()

	body, err := json.Marshal(map[string]any{
		"id":      messageID.String(),
		"type":    "lot.created",
		"version": 1,
		"data": map[string]any{
			"lot_id":       lotID.String(),
			"farm":         map[string]any{"id": uuid.NewString(), "name": "La Esperanza", "altitude": 1800},
			"producer":     map[string]any{"id": uuid.NewString(), "name": "Maria Lopez"},
			"custodian_id": uuid.NewString(),
			"code":         "LT-001",
			"variety":      "Castillo",
			"process":      "washed",
			"harvest_date": "2026-01-10",
			"stage":        stage,
			"status":       "active",
		},
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	return h.lotCreated.Handle(context.Background(), libmessaging.Delivery{
		ID:         messageID.String(),
		Type:       "lot.created",
		Version:    1,
		OccurredAt: time.Now().UTC(),
		Body:       body,
	})
}

func (h harness) get(path string, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.mux.ServeHTTP(rec, req)
	return rec
}

func TestLotCreatedEventFeedsPublicLot(t *testing.T) {
	h := newHarness(t)
	lotID := uuid.New()

	if err := h.publish(t, uuid.New(), lotID, "farm"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	rec := h.get("/public/lots/"+lotID.String(), nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Code        string `json:"code"`
		Stage       string `json:"stage"`
		QRAvailable bool   `json:"qr_available"`
		Farm        struct {
			Name string `json:"name"`
		} `json:"farm"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if body.Code != "LT-001" || body.Stage != "farm" || body.Farm.Name != "La Esperanza" || body.QRAvailable {
		t.Errorf("unexpected body %+v", body)
	}
}

func TestDuplicateEventIsIdempotent(t *testing.T) {
	h := newHarness(t)
	messageID, lotID := uuid.New(), uuid.New()

	for i := 0; i < 2; i++ {
		if err := h.publish(t, messageID, lotID, "none"); err != nil {
			t.Fatalf("delivery %d: unexpected error: %v", i, err)
		}
	}

	if rec := h.get("/public/lots/"+lotID.String(), nil); rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestInvalidEventIsPermanentFailure(t *testing.T) {
	h := newHarness(t)

	err := h.publish(t, uuid.New(), uuid.Nil, "none")

	if !libmessaging.IsPermanent(err) {
		t.Fatalf("expected permanent error, got %v", err)
	}
}

func TestUnknownLotIsNotFound(t *testing.T) {
	h := newHarness(t)
	id := uuid.NewString()

	if rec := h.get("/public/lots/"+id, nil); rec.Code != http.StatusNotFound {
		t.Errorf("lot: expected 404, got %d", rec.Code)
	}
	if rec := h.get("/l/"+id, nil); rec.Code != http.StatusNotFound {
		t.Errorf("scan: expected 404, got %d", rec.Code)
	}
}

func TestScanRedirectsToLotPage(t *testing.T) {
	h := newHarness(t)
	lotID := uuid.New()
	if err := h.publish(t, uuid.New(), lotID, "packaging"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	rec := h.get("/l/"+lotID.String(), nil)

	if rec.Code != http.StatusMovedPermanently || rec.Header().Get("Location") != "/lotes/"+lotID.String() {
		t.Fatalf("unexpected redirect %d %q", rec.Code, rec.Header().Get("Location"))
	}
}

func TestQRImageOnlyForPackagedLots(t *testing.T) {
	h := newHarness(t)
	packaged, early := uuid.New(), uuid.New()
	if err := h.publish(t, uuid.New(), packaged, "packaging"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := h.publish(t, uuid.New(), early, "roasting"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	actor := map[string]string{controllers.ActorIDHeader: uuid.NewString()}

	if rec := h.get("/public/lots/"+packaged.String()+"/qr", nil); rec.Code != http.StatusUnauthorized {
		t.Errorf("without actor: expected 401, got %d", rec.Code)
	}

	rec := h.get("/public/lots/"+packaged.String()+"/qr?size=256&format=png", actor)
	if rec.Code != http.StatusOK || rec.Header().Get("Content-Type") != "image/png" {
		t.Fatalf("packaged: expected png, got %d %q", rec.Code, rec.Header().Get("Content-Type"))
	}
	if _, err := png.Decode(bytes.NewReader(rec.Body.Bytes())); err != nil {
		t.Errorf("invalid png: %v", err)
	}

	svg := h.get("/public/lots/"+packaged.String()+"/qr?format=svg", actor)
	if svg.Code != http.StatusOK || svg.Header().Get("Content-Type") != "image/svg+xml" {
		t.Errorf("svg: unexpected response %d %q", svg.Code, svg.Header().Get("Content-Type"))
	}

	if rec := h.get("/public/lots/"+early.String()+"/qr", actor); rec.Code != http.StatusConflict {
		t.Errorf("early stage: expected 409, got %d", rec.Code)
	}
	if rec := h.get("/public/lots/"+packaged.String()+"/qr?size=300", actor); rec.Code != http.StatusBadRequest {
		t.Errorf("invalid size: expected 400, got %d", rec.Code)
	}
}

func TestPackagedLotAdvertisesQR(t *testing.T) {
	h := newHarness(t)
	lotID := uuid.New()
	if err := h.publish(t, uuid.New(), lotID, "packaging"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	rec := h.get("/public/lots/"+lotID.String(), nil)

	var body struct {
		QRAvailable bool `json:"qr_available"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil || !body.QRAvailable {
		t.Errorf("expected qr_available, got %v %v", body, err)
	}
}
