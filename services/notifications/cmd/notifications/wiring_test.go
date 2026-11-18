package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/google/uuid"

	libmessaging "rutadeorigen/lib/messaging"
	inputmessaging "rutadeorigen/notifications/internal/infrastructure/input/messaging"
	"rutadeorigen/notifications/internal/infrastructure/output/external/emailrender"
	"rutadeorigen/notifications/internal/infrastructure/output/external/resend"
)

type resendFake struct {
	mu       sync.Mutex
	status   int
	requests []map[string]any
	keys     []string
}

func (f *resendFake) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var body map[string]any
	_ = json.NewDecoder(r.Body).Decode(&body)

	f.mu.Lock()
	f.requests = append(f.requests, body)
	f.keys = append(f.keys, r.Header.Get("Idempotency-Key"))
	status := f.status
	f.mu.Unlock()

	w.WriteHeader(status)
	_, _ = io.WriteString(w, `{"id":"email-1","message":"detail"}`)
}

func (f *resendFake) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.requests)
}

func newHandler(t *testing.T, status int) (*inputmessaging.EvidenceResultHandler, *resendFake) {
	t.Helper()

	fake := &resendFake{status: status}
	server := httptest.NewServer(fake)
	t.Cleanup(server.Close)

	renderer, err := emailrender.New("https://rutadeorigen.co")
	if err != nil {
		t.Fatal(err)
	}
	sender := resend.NewWithClient(resend.Config{APIKey: "re_test", From: "noreply@rutadeorigen.co", BaseURL: server.URL},
		server.Client(), slog.New(slog.NewJSONHandler(io.Discard, nil)))

	return newEvidenceResultHandler(nil, renderer, sender), fake
}

func resultEvent(eventType, email string) libmessaging.Delivery {
	body := fmt.Sprintf(`{"id":"env","type":%q,"version":1,"correlation_id":"corr","data":{"evidence_id":%q,"lot_id":%q,"lot_code":"LT-001","stage":"farm","file_name":"foto.jpg","reason":"No es una imagen","uploader":{"actor_id":%q,"role":"producer","name":"Maria","email":%q}}}`,
		eventType, uuid.New(), uuid.New(), uuid.New(), email)

	return libmessaging.Delivery{ID: uuid.NewString(), Type: eventType, Body: []byte(body)}
}

func TestSendsAnEmailForProcessedEvidence(t *testing.T) {
	handler, fake := newHandler(t, http.StatusOK)
	event := resultEvent("evidence.processed", "maria@example.com")

	if err := handler.Handle(context.Background(), event); err != nil {
		t.Fatal(err)
	}

	if fake.count() != 1 {
		t.Fatalf("want one email, got %d", fake.count())
	}
	if fake.requests[0]["subject"] != "Tu foto del lote LT-001 ya está publicada" {
		t.Errorf("subject = %v", fake.requests[0]["subject"])
	}
	if fake.keys[0] == "" {
		t.Error("the idempotency key must be sent")
	}
}

func TestSendsAnEmailForRejectedEvidence(t *testing.T) {
	handler, fake := newHandler(t, http.StatusOK)

	if err := handler.Handle(context.Background(), resultEvent("evidence.rejected", "maria@example.com")); err != nil {
		t.Fatal(err)
	}

	if fake.count() != 1 || fake.requests[0]["subject"] != "No pudimos procesar tu foto del lote LT-001" {
		t.Errorf("unexpected requests %v", fake.requests)
	}
}

func TestRedeliveredMessageDoesNotSendAgain(t *testing.T) {
	handler, fake := newHandler(t, http.StatusOK)
	event := resultEvent("evidence.processed", "maria@example.com")

	for range 2 {
		if err := handler.Handle(context.Background(), event); err != nil {
			t.Fatal(err)
		}
	}

	if fake.count() != 1 {
		t.Errorf("want one email, got %d", fake.count())
	}
}

func TestProviderOutageIsRetryable(t *testing.T) {
	handler, _ := newHandler(t, http.StatusServiceUnavailable)

	err := handler.Handle(context.Background(), resultEvent("evidence.processed", "maria@example.com"))

	if err == nil || libmessaging.IsPermanent(err) {
		t.Errorf("want a retryable error, got %v", err)
	}
}

func TestProviderRejectionIsPermanent(t *testing.T) {
	handler, _ := newHandler(t, http.StatusUnprocessableEntity)

	err := handler.Handle(context.Background(), resultEvent("evidence.processed", "maria@example.com"))

	if !libmessaging.IsPermanent(err) {
		t.Errorf("want a permanent error, got %v", err)
	}
}

func TestInvalidRecipientIsPermanentAndSendsNothing(t *testing.T) {
	handler, fake := newHandler(t, http.StatusOK)

	err := handler.Handle(context.Background(), resultEvent("evidence.processed", "not-an-email"))

	if !libmessaging.IsPermanent(err) || fake.count() != 0 {
		t.Errorf("err=%v emails=%d", err, fake.count())
	}
}
