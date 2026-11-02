package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"rutadeorigen/processor/internal/application/output/ports"
	"rutadeorigen/processor/internal/platform/exception"
)

type recordingTx struct {
	pgx.Tx
	sql          string
	args         []any
	rowsAffected int64
	exists       bool
	err          error
}

func (r *recordingTx) Exec(_ context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	r.sql, r.args = sql, args
	if r.err != nil {
		return pgconn.CommandTag{}, r.err
	}
	return pgconn.NewCommandTag("INSERT 0 " + string(rune('0'+r.rowsAffected))), nil
}

func (r *recordingTx) QueryRow(_ context.Context, sql string, args ...any) pgx.Row {
	r.sql, r.args = sql, args
	return fixedRow{exists: r.exists, err: r.err}
}

type fixedRow struct {
	exists bool
	err    error
}

func (f fixedRow) Scan(dest ...any) error {
	if f.err != nil {
		return f.err
	}
	*(dest[0].(*bool)) = f.exists
	return nil
}

func insideTx(tx *recordingTx) context.Context {
	return context.WithValue(context.Background(), txKey{}, pgx.Tx(tx))
}

func TestRegisterReportsFirstTime(t *testing.T) {
	tx := &recordingTx{rowsAffected: 1}
	messageID := uuid.New()

	first, err := (&ProcessedEventRepository{}).Register(insideTx(tx), messageID, "evidence.uploaded")

	if err != nil || !first {
		t.Fatalf("want first registration, got %v %v", first, err)
	}
	if tx.args[0] != messageID || tx.args[1] != "evidence.uploaded" {
		t.Errorf("unexpected args %v", tx.args)
	}
}

func TestRegisterReportsRepeatedMessage(t *testing.T) {
	tx := &recordingTx{rowsAffected: 0}

	first, err := (&ProcessedEventRepository{}).Register(insideTx(tx), uuid.New(), "evidence.uploaded")

	if err != nil || first {
		t.Errorf("want repeated message, got %v %v", first, err)
	}
}

func TestRegisterWrapsDatabaseError(t *testing.T) {
	tx := &recordingTx{err: errors.New("db down")}

	_, err := (&ProcessedEventRepository{}).Register(insideTx(tx), uuid.New(), "evidence.uploaded")

	if !exception.HasCode(err, exception.CodeInternal) {
		t.Errorf("want internal error, got %v", err)
	}
}

func TestExistsReadsFlag(t *testing.T) {
	for _, want := range []bool{true, false} {
		tx := &recordingTx{exists: want}

		got, err := (&ProcessedEventRepository{}).Exists(insideTx(tx), uuid.New())

		if err != nil || got != want {
			t.Errorf("want %v, got %v %v", want, got, err)
		}
	}
}

func TestExistsWrapsDatabaseError(t *testing.T) {
	tx := &recordingTx{err: errors.New("db down")}

	_, err := (&ProcessedEventRepository{}).Exists(insideTx(tx), uuid.New())

	if !exception.HasCode(err, exception.CodeInternal) {
		t.Errorf("want internal error, got %v", err)
	}
}

func TestOutboxPublisherStoresEnvelope(t *testing.T) {
	tx := &recordingTx{rowsAffected: 1}
	event := ports.Event{
		ID:            uuid.New(),
		Type:          ports.EventEvidenceProcessed,
		AggregateType: ports.AggregateEvidence,
		AggregateID:   uuid.New(),
		Version:       1,
		OccurredAt:    time.Date(2026, time.November, 2, 6, 0, 0, 0, time.UTC),
		ActorID:       uuid.New(),
		ActorRole:     "producer",
		CorrelationID: "req-1",
		Payload:       map[string]string{"evidence_id": "abc"},
	}

	if err := (&OutboxPublisher{}).Publish(insideTx(tx), event); err != nil {
		t.Fatal(err)
	}

	var stored struct {
		Type          string            `json:"type"`
		CorrelationID string            `json:"correlation_id"`
		Data          map[string]string `json:"data"`
	}
	if err := json.Unmarshal(tx.args[5].([]byte), &stored); err != nil {
		t.Fatal(err)
	}
	if stored.Type != "evidence.processed" || stored.CorrelationID != "req-1" || stored.Data["evidence_id"] != "abc" {
		t.Errorf("unexpected envelope %+v", stored)
	}
}

func TestOutboxPublisherWrapsDatabaseError(t *testing.T) {
	tx := &recordingTx{err: errors.New("db down")}

	err := (&OutboxPublisher{}).Publish(insideTx(tx), ports.Event{ID: uuid.New(), Payload: "x"})

	if !exception.HasCode(err, exception.CodeInternal) {
		t.Errorf("want internal error, got %v", err)
	}
}
