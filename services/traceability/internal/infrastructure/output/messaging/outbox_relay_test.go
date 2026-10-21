package messaging

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func sampleRow(payload string) outboxRow {
	return outboxRow{
		ID:        uuid.New(),
		EventType: "lot.created",
		Version:   3,
		Payload:   []byte(payload),
		CreatedAt: time.Date(2026, time.October, 21, 15, 0, 0, 0, time.UTC),
	}
}

func TestToMessageMapsTheRow(t *testing.T) {
	row := sampleRow(`{"id":"x","correlation_id":"req-123","data":{}}`)

	got := row.toMessage()

	if got.ID != row.ID.String() {
		t.Errorf("ID: got %q, want %q", got.ID, row.ID.String())
	}
	if got.Type != "lot.created" || got.Version != 3 {
		t.Errorf("unexpected type %q or version %d", got.Type, got.Version)
	}
	if !got.OccurredAt.Equal(row.CreatedAt) {
		t.Errorf("OccurredAt: got %v, want %v", got.OccurredAt, row.CreatedAt)
	}
	if string(got.Body) != string(row.Payload) {
		t.Error("expected the payload as body")
	}
	if got.CorrelationID != "req-123" {
		t.Errorf("CorrelationID: got %q, want %q", got.CorrelationID, "req-123")
	}
}

func TestToMessageWithoutCorrelationID(t *testing.T) {
	got := sampleRow(`{"id":"x"}`).toMessage()

	if got.CorrelationID != "" {
		t.Errorf("got %q, want empty", got.CorrelationID)
	}
}

func TestToMessageWithAnInvalidPayloadKeepsTheBody(t *testing.T) {
	row := sampleRow(`not json`)

	got := row.toMessage()

	if got.CorrelationID != "" {
		t.Errorf("got %q, want empty", got.CorrelationID)
	}
	if string(got.Body) != "not json" || got.ID != row.ID.String() {
		t.Error("expected the row data to be kept")
	}
}
