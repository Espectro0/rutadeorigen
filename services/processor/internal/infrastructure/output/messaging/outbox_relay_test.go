package messaging

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestToMessageMapsTheRow(t *testing.T) {
	row := outboxRow{
		ID:        uuid.New(),
		EventType: "evidence.processed",
		Version:   1,
		Payload:   []byte(`{"id":"x","correlation_id":"req-123","data":{}}`),
		CreatedAt: time.Date(2026, time.November, 2, 6, 0, 0, 0, time.UTC),
	}

	message := row.toMessage()

	if message.ID != row.ID.String() || message.Type != "evidence.processed" || message.Version != 1 {
		t.Errorf("unexpected message %+v", message)
	}
	if message.CorrelationID != "req-123" {
		t.Errorf("correlation id = %q", message.CorrelationID)
	}
	if !message.OccurredAt.Equal(row.CreatedAt) || string(message.Body) != string(row.Payload) {
		t.Error("time and body must be carried over")
	}
}

func TestToMessageToleratesPayloadWithoutCorrelation(t *testing.T) {
	row := outboxRow{ID: uuid.New(), Payload: []byte(`not json`)}

	if row.toMessage().CorrelationID != "" {
		t.Error("unreadable payload must give an empty correlation id")
	}
}
