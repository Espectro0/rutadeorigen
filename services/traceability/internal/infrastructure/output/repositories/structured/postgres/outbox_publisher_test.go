package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"rutadeorigen/traceability/internal/application/output/ports"
	"rutadeorigen/traceability/internal/platform/exception"
)

type recordingTx struct {
	pgx.Tx
	sql  string
	args []any
	err  error
}

func (r *recordingTx) Exec(_ context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	r.sql, r.args = sql, args
	return pgconn.CommandTag{}, r.err
}

func sampleEvent() ports.Event {
	return ports.Event{
		ID:            uuid.New(),
		Type:          ports.EventLotCreated,
		AggregateType: ports.AggregateLot,
		AggregateID:   uuid.New(),
		Version:       1,
		OccurredAt:    time.Date(2026, time.October, 21, 15, 0, 0, 0, time.UTC),
		ActorID:       uuid.New(),
		ActorRole:     "producer",
		CorrelationID: "req-123",
		Payload:       map[string]string{"code": "LT-001"},
	}
}

func publish(t *testing.T, tx *recordingTx, e ports.Event) error {
	t.Helper()
	ctx := context.WithValue(context.Background(), txKey{}, pgx.Tx(tx))
	return NewOutboxPublisher(nil).Publish(ctx, e)
}

func TestOutboxPublisherInsertsTheEventInTheCurrentTransaction(t *testing.T) {
	tx := &recordingTx{}
	e := sampleEvent()

	if err := publish(t, tx, e); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if tx.sql != insertOutbox {
		t.Fatal("expected the outbox insert")
	}
	if len(tx.args) != 7 {
		t.Fatalf("got %d args, want 7", len(tx.args))
	}
	if tx.args[0] != e.ID || tx.args[2] != e.AggregateID {
		t.Error("unexpected event or aggregate ID")
	}
	if fmt.Sprint(tx.args[1]) != "lot" || fmt.Sprint(tx.args[3]) != "lot.created" {
		t.Errorf("unexpected aggregate type %v or event type %v", tx.args[1], tx.args[3])
	}
	if tx.args[4] != 1 || tx.args[6] != e.OccurredAt {
		t.Error("unexpected version or date")
	}
}

func TestOutboxPublisherWritesASnakeCaseEnvelope(t *testing.T) {
	tx := &recordingTx{}
	e := sampleEvent()

	if err := publish(t, tx, e); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	body, ok := tx.args[5].([]byte)
	if !ok {
		t.Fatalf("unexpected payload type %T", tx.args[5])
	}
	var got map[string]any
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	want := map[string]any{
		"id":             e.ID.String(),
		"type":           "lot.created",
		"version":        float64(1),
		"occurred_at":    "2026-10-21T15:00:00Z",
		"actor_id":       e.ActorID.String(),
		"actor_role":     "producer",
		"correlation_id": "req-123",
	}
	for key, value := range want {
		if got[key] != value {
			t.Errorf("%s: got %v, want %v", key, got[key], value)
		}
	}
	data, ok := got["data"].(map[string]any)
	if !ok || data["code"] != "LT-001" {
		t.Errorf("unexpected data %v", got["data"])
	}
	if len(got) != len(want)+1 {
		t.Errorf("got %d keys, want %d", len(got), len(want)+1)
	}
}

func TestOutboxPublisherFailsWhenTheInsertFails(t *testing.T) {
	cause := errors.New("connection reset")
	tx := &recordingTx{err: cause}

	err := publish(t, tx, sampleEvent())

	e, ok := exception.From(err)
	if !ok || e.Kind() != exception.KindInternal {
		t.Errorf("expected an internal error, got %v", err)
	}
	if !errors.Is(err, cause) {
		t.Error("expected the original cause to be kept")
	}
}

func TestOutboxPublisherFailsWhenThePayloadIsNotJSON(t *testing.T) {
	tx := &recordingTx{}
	e := sampleEvent()
	e.Payload = make(chan int)

	err := publish(t, tx, e)

	ex, ok := exception.From(err)
	if !ok || ex.Kind() != exception.KindInternal {
		t.Errorf("expected an internal error, got %v", err)
	}
	if tx.sql != "" {
		t.Error("nothing must be inserted when the payload cannot be encoded")
	}
}
