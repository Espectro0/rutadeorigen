package messaging_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"

	"rutadeorigen/traceability/internal/application/output/ports"
	"rutadeorigen/traceability/internal/infrastructure/output/messaging"
)

type samplePayload struct {
	Code string `json:"code"`
}

func TestLogPublisherWritesTheEvent(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&buf, nil))

	event := ports.Event{
		ID:            uuid.New(),
		Type:          ports.EventLotCreated,
		AggregateType: ports.AggregateLot,
		AggregateID:   uuid.New(),
		Version:       1,
		OccurredAt:    time.Date(2026, time.October, 10, 15, 0, 0, 0, time.UTC),
		ActorID:       uuid.New(),
		ActorRole:     "producer",
		CorrelationID: "req-123",
		Payload:       samplePayload{Code: "LOT-001"},
	}

	if err := messaging.NewLogPublisher(log).Publish(context.Background(), event); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("log line is not valid JSON: %v\n%s", err, buf.String())
	}

	want := map[string]any{
		"msg":            "event published",
		"level":          "INFO",
		"event_id":       event.ID.String(),
		"event_type":     "lot.created",
		"aggregate_type": "lot",
		"aggregate_id":   event.AggregateID.String(),
		"version":        float64(1),
		"occurred_at":    "2026-10-10T15:00:00Z",
		"actor_id":       event.ActorID.String(),
		"actor_role":     "producer",
		"correlation_id": "req-123",
	}
	for key, value := range want {
		if entry[key] != value {
			t.Errorf("%s: got %v, want %v", key, entry[key], value)
		}
	}

	payload, ok := entry["payload"].(map[string]any)
	if !ok || payload["code"] != "LOT-001" {
		t.Errorf("payload must be serialized as JSON, got %v", entry["payload"])
	}
}
