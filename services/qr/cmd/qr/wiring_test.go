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

	return h.publishWithCustodian(t, messageID, lotID, stage, uuid.New())
}

func (h harness) publishWithCustodian(t *testing.T, messageID uuid.UUID, lotID uuid.UUID, stage string, custodianID uuid.UUID) error {
	t.Helper()

	body, err := json.Marshal(map[string]any{
		"id":      messageID.String(),
		"type":    "lot.created",
		"version": 1,
		"data": map[string]any{
			"lot_id":       lotID.String(),
			"farm":         map[string]any{"id": uuid.NewString(), "name": "La Esperanza", "altitude": 1800},
			"producer":     map[string]any{"id": uuid.NewString(), "name": "Maria Lopez"},
			"custodian_id": custodianID.String(),
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

	if rec.Code != http.StatusFound || rec.Header().Get("Location") != "/lotes/"+lotID.String() {
		t.Fatalf("unexpected redirect %d %q", rec.Code, rec.Header().Get("Location"))
	}
}

func (h harness) registerActor(t *testing.T, userID, role string) (uuid.UUID, map[string]string) {
	t.Helper()

	id := uuid.New()
	h.publishActor(t, userID, id, role)
	return id, map[string]string{controllers.ActorIDHeader: id.String()}
}

func TestQRImageOnlyForPackagedLots(t *testing.T) {
	h := newHarness(t)
	brandID, actor := h.registerActor(t, "brand_user", "brand")
	packaged, early := uuid.New(), uuid.New()
	if err := h.publishWithCustodian(t, uuid.New(), packaged, "packaging", brandID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := h.publishWithCustodian(t, uuid.New(), early, "roasting", brandID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

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

func (h harness) publishActor(t *testing.T, userID string, actorID uuid.UUID, role string) {
	t.Helper()

	messageID := uuid.New()
	body, err := json.Marshal(map[string]any{
		"id":      messageID.String(),
		"type":    "actor.registered",
		"version": 1,
		"data":    map[string]any{"actor_id": actorID.String(), "user_id": userID, "type": role},
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	err = h.lotCreated.Handle(context.Background(), libmessaging.Delivery{
		ID:         messageID.String(),
		Type:       "actor.registered",
		Version:    1,
		OccurredAt: time.Now().UTC(),
		Body:       body,
	})
	if err != nil {
		t.Fatalf("publish actor: %v", err)
	}
}

func (h harness) publishLotEvent(t *testing.T, eventType string, lotID uuid.UUID, version int, data map[string]any) error {
	t.Helper()

	messageID := uuid.New()
	data["lot_id"] = lotID.String()
	body, err := json.Marshal(map[string]any{
		"id":      messageID.String(),
		"type":    eventType,
		"version": version,
		"data":    data,
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	return h.lotCreated.Handle(context.Background(), libmessaging.Delivery{
		ID:         messageID.String(),
		Type:       eventType,
		Version:    version,
		OccurredAt: time.Now().UTC(),
		Body:       body,
	})
}

func TestLotVoidedEventBlocksTheQRImage(t *testing.T) {
	h := newHarness(t)
	brandID, actor := h.registerActor(t, "brand_user", "brand")
	lotID := uuid.New()
	if err := h.publishWithCustodian(t, uuid.New(), lotID, "packaging", brandID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	path := "/public/lots/" + lotID.String() + "/qr?format=png"
	if rec := h.get(path, actor); rec.Code != http.StatusOK {
		t.Fatalf("before the void: expected 200, got %d", rec.Code)
	}

	err := h.publishLotEvent(t, "lot.voided", lotID, 2, map[string]any{"reason": "Lote contaminado", "status": "voided"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rec := h.get(path, actor); rec.Code != http.StatusConflict {
		t.Errorf("after the void: expected 409, got %d", rec.Code)
	}
}

func TestLotTransferredEventIsProjected(t *testing.T) {
	h := newHarness(t)
	lotID := uuid.New()
	if err := h.publish(t, uuid.New(), lotID, "packaging"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err := h.publishLotEvent(t, "lot.transferred", lotID, 2, map[string]any{"custodian_id": uuid.NewString(), "stage": "packaging"})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec := h.get("/public/lots/"+lotID.String(), nil); rec.Code != http.StatusOK {
		t.Errorf("the lot must still be readable, got %d", rec.Code)
	}
}

func TestTransferOfAnUnknownLotIsRetried(t *testing.T) {
	h := newHarness(t)

	err := h.publishLotEvent(t, "lot.transferred", uuid.New(), 2, map[string]any{"custodian_id": uuid.NewString()})

	if err == nil || libmessaging.IsPermanent(err) {
		t.Errorf("expected a retryable error until lot.created arrives, got %v", err)
	}
}

func TestQRImageIsDeniedToEveryoneButTheCustodianBrand(t *testing.T) {
	h := newHarness(t)
	brandID, brand := h.registerActor(t, "brand_user", "brand")
	_, otherBrand := h.registerActor(t, "other_brand", "brand")
	_, producer := h.registerActor(t, "producer_user", "producer")
	_, organization := h.registerActor(t, "org_user", "organization")
	unknown := map[string]string{controllers.ActorIDHeader: uuid.NewString()}
	lotID := uuid.New()
	if err := h.publishWithCustodian(t, uuid.New(), lotID, "packaging", brandID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	path := "/public/lots/" + lotID.String() + "/qr?format=png"

	if rec := h.get(path, brand); rec.Code != http.StatusOK {
		t.Errorf("custodian brand: expected 200, got %d", rec.Code)
	}
	for name, headers := range map[string]map[string]string{
		"brand that is not the custodian": otherBrand,
		"producer":                        producer,
		"organization":                    organization,
		"actor without projection":        unknown,
	} {
		if rec := h.get(path, headers); rec.Code != http.StatusForbidden {
			t.Errorf("%s: expected 403, got %d", name, rec.Code)
		}
	}
}

func TestQRImageFollowsTheCustodianAfterATransfer(t *testing.T) {
	h := newHarness(t)
	firstID, first := h.registerActor(t, "first_brand", "brand")
	secondID, second := h.registerActor(t, "second_brand", "brand")
	lotID := uuid.New()
	if err := h.publishWithCustodian(t, uuid.New(), lotID, "packaging", firstID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	path := "/public/lots/" + lotID.String() + "/qr?format=png"
	if rec := h.get(path, first); rec.Code != http.StatusOK {
		t.Fatalf("before the transfer: expected 200, got %d", rec.Code)
	}

	err := h.publishLotEvent(t, "lot.transferred", lotID, 2, map[string]any{"custodian_id": secondID.String(), "stage": "packaging"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rec := h.get(path, first); rec.Code != http.StatusForbidden {
		t.Errorf("previous custodian: expected 403, got %d", rec.Code)
	}
	if rec := h.get(path, second); rec.Code != http.StatusOK {
		t.Errorf("new custodian: expected 200, got %d", rec.Code)
	}
}

type statsBody struct {
	LotID       string  `json:"lot_id"`
	Total       int     `json:"total"`
	FirstScanAt *string `json:"first_scan_at"`
	LastScanAt  *string `json:"last_scan_at"`
	Days        int     `json:"days"`
	Daily       []struct {
		Date  string `json:"date"`
		Count int    `json:"count"`
	} `json:"daily"`
}

func TestScanStatsAreOnlyForTheOrganization(t *testing.T) {
	h := newHarness(t)
	_, organization := h.registerActor(t, "org_user", "organization")
	brandID, brand := h.registerActor(t, "brand_user", "brand")
	_, producer := h.registerActor(t, "producer_user", "producer")
	lotID := uuid.New()
	if err := h.publishWithCustodian(t, uuid.New(), lotID, "packaging", brandID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for i := 0; i < 3; i++ {
		if rec := h.get("/l/"+lotID.String(), nil); rec.Code != http.StatusFound {
			t.Fatalf("scan %d: expected 302, got %d", i, rec.Code)
		}
	}
	path := "/lots/" + lotID.String() + "/scans/stats"

	if rec := h.get(path, nil); rec.Code != http.StatusUnauthorized {
		t.Errorf("without actor: expected 401, got %d", rec.Code)
	}
	for name, headers := range map[string]map[string]string{"brand custodian": brand, "producer": producer} {
		if rec := h.get(path, headers); rec.Code != http.StatusForbidden {
			t.Errorf("%s: expected 403, got %d", name, rec.Code)
		}
	}

	rec := h.get(path, organization)
	if rec.Code != http.StatusOK {
		t.Fatalf("organization: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var body statsBody
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if body.LotID != lotID.String() || body.Total != 3 || body.Days != 30 || body.FirstScanAt == nil || body.LastScanAt == nil {
		t.Errorf("unexpected body %+v", body)
	}
	if len(body.Daily) != 1 || body.Daily[0].Count != 3 {
		t.Errorf("expected the three scans on one day, got %+v", body.Daily)
	}
}

func TestScanStatsOfALotWithoutScans(t *testing.T) {
	h := newHarness(t)
	_, organization := h.registerActor(t, "org_user", "organization")
	lotID := uuid.New()
	if err := h.publish(t, uuid.New(), lotID, "farm"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	rec := h.get("/lots/"+lotID.String()+"/scans/stats?days=7", organization)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var body statsBody
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if body.Total != 0 || body.Days != 7 || body.FirstScanAt != nil || body.LastScanAt != nil || body.Daily == nil || len(body.Daily) != 0 {
		t.Errorf("unexpected body %+v", body)
	}
}

func TestScanStatsValidateDaysAndLot(t *testing.T) {
	h := newHarness(t)
	_, organization := h.registerActor(t, "org_user", "organization")
	lotID := uuid.New()
	if err := h.publish(t, uuid.New(), lotID, "farm"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rec := h.get("/lots/"+lotID.String()+"/scans/stats?days=0", organization); rec.Code != http.StatusBadRequest {
		t.Errorf("invalid days: expected 400, got %d", rec.Code)
	}
	if rec := h.get("/lots/"+lotID.String()+"/scans/stats?days=400", organization); rec.Code != http.StatusBadRequest {
		t.Errorf("too many days: expected 400, got %d", rec.Code)
	}
	if rec := h.get("/lots/"+uuid.NewString()+"/scans/stats", organization); rec.Code != http.StatusNotFound {
		t.Errorf("unknown lot: expected 404, got %d", rec.Code)
	}
}
