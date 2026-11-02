package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"rutadeorigen/notifications/internal/domain/notification"
	"rutadeorigen/notifications/internal/platform/exception"
)

type recordingTx struct {
	pgx.Tx
	execs        []string
	args         [][]any
	rowsAffected int64
	exists       bool
	row          func(dest ...any) error
	err          error
}

func (r *recordingTx) Exec(_ context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	r.execs = append(r.execs, sql)
	r.args = append(r.args, args)
	if r.err != nil {
		return pgconn.CommandTag{}, r.err
	}
	return pgconn.NewCommandTag("UPDATE " + string(rune('0'+r.rowsAffected))), nil
}

func (r *recordingTx) QueryRow(_ context.Context, _ string, _ ...any) pgx.Row {
	return fixedRow{exists: r.exists, err: r.err, fill: r.row}
}

type fixedRow struct {
	exists bool
	err    error
	fill   func(dest ...any) error
}

func (f fixedRow) Scan(dest ...any) error {
	if f.err != nil {
		return f.err
	}
	if f.fill != nil {
		return f.fill(dest...)
	}
	*(dest[0].(*bool)) = f.exists
	return nil
}

func insideTx(tx *recordingTx) context.Context {
	return context.WithValue(context.Background(), txKey{}, pgx.Tx(tx))
}

func candidate(t *testing.T) *notification.Notification {
	t.Helper()

	recipient, err := notification.NewRecipient("Maria", "maria@example.com")
	if err != nil {
		t.Fatal(err)
	}
	n, err := notification.New(notification.NewParams{
		ID:         uuid.New(),
		EvidenceID: uuid.New(),
		Kind:       notification.KindEvidenceProcessed,
		Recipient:  recipient,
		CreatedAt:  time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}
	return n
}

func TestRegisterReportsFirstTime(t *testing.T) {
	tx := &recordingTx{rowsAffected: 1}
	messageID := uuid.New()

	first, err := (&ProcessedEventRepository{}).Register(insideTx(tx), messageID, "evidence.processed")

	if err != nil || !first {
		t.Fatalf("want first registration, got %v %v", first, err)
	}
	if tx.args[0][0] != messageID || tx.args[0][1] != "evidence.processed" {
		t.Errorf("unexpected args %v", tx.args[0])
	}
}

func TestRegisterReportsRepeatedMessage(t *testing.T) {
	first, err := (&ProcessedEventRepository{}).Register(insideTx(&recordingTx{rowsAffected: 0}), uuid.New(), "evidence.processed")

	if err != nil || first {
		t.Errorf("want repeated message, got %v %v", first, err)
	}
}

func TestExistsReadsTheFlag(t *testing.T) {
	found, err := (&ProcessedEventRepository{}).Exists(insideTx(&recordingTx{exists: true}), uuid.New())

	if err != nil || !found {
		t.Errorf("want found, got %v %v", found, err)
	}
}

func TestRepositoriesWrapDatabaseErrors(t *testing.T) {
	tx := &recordingTx{err: errors.New("db down")}

	if _, err := (&ProcessedEventRepository{}).Exists(insideTx(tx), uuid.New()); !exception.HasCode(err, exception.CodeInternal) {
		t.Errorf("exists: want internal error, got %v", err)
	}
	if _, err := (&ProcessedEventRepository{}).Register(insideTx(tx), uuid.New(), "x"); !exception.HasCode(err, exception.CodeInternal) {
		t.Errorf("register: want internal error, got %v", err)
	}
	if _, err := (&NotificationLogRepository{}).Reserve(insideTx(tx), candidate(t)); !exception.HasCode(err, exception.CodeInternal) {
		t.Errorf("reserve: want internal error, got %v", err)
	}
	if err := (&NotificationLogRepository{}).Update(insideTx(tx), candidate(t)); !exception.HasCode(err, exception.CodeInternal) {
		t.Errorf("update: want internal error, got %v", err)
	}
}

func TestReserveRestoresTheStoredNotification(t *testing.T) {
	stored := candidate(t)
	createdAt := stored.CreatedAt()
	tx := &recordingTx{row: func(dest ...any) error {
		*(dest[0].(*uuid.UUID)) = stored.ID()
		*(dest[1].(*uuid.UUID)) = stored.EvidenceID()
		*(dest[2].(*string)) = "evidence_processed"
		*(dest[3].(*string)) = "Maria"
		*(dest[4].(*string)) = "maria@example.com"
		*(dest[5].(*string)) = "pending"
		*(dest[6].(*int)) = 1
		*(dest[7].(*string)) = "timeout"
		*(dest[8].(*time.Time)) = createdAt
		return nil
	}}

	got, err := (&NotificationLogRepository{}).Reserve(insideTx(tx), stored)

	if err != nil {
		t.Fatal(err)
	}
	if got.ID() != stored.ID() || !got.IsPending() || got.Attempts() != 1 || got.LastError() != "timeout" {
		t.Errorf("unexpected notification %+v", got)
	}
	if got.Recipient().Email() != "maria@example.com" {
		t.Errorf("unexpected recipient %q", got.Recipient().Email())
	}
}

func TestUpdateRejectsUnknownNotification(t *testing.T) {
	err := (&NotificationLogRepository{}).Update(insideTx(&recordingTx{rowsAffected: 0}), candidate(t))

	if !exception.HasCode(err, exception.CodeInternal) {
		t.Errorf("want internal error, got %v", err)
	}
}

func TestUpdateStoresSentAtOnlyWhenSent(t *testing.T) {
	pending := candidate(t)
	tx := &recordingTx{rowsAffected: 1}
	if err := (&NotificationLogRepository{}).Update(insideTx(tx), pending); err != nil {
		t.Fatal(err)
	}
	if got := tx.args[0][4].(*time.Time); got != nil {
		t.Error("a pending notification must store a null sent_at")
	}

	sent, _ := pending.MarkSent(time.Now())
	tx = &recordingTx{rowsAffected: 1}
	if err := (&NotificationLogRepository{}).Update(insideTx(tx), sent); err != nil {
		t.Fatal(err)
	}
	if got := tx.args[0][4].(*time.Time); got == nil {
		t.Error("a sent notification must store its sent_at")
	}
}
