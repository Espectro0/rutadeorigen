package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"rutadeorigen/qr/internal/domain/lotview"
	"rutadeorigen/qr/internal/domain/qrcode"
	"rutadeorigen/qr/internal/domain/scan"
	"rutadeorigen/qr/internal/platform/exception"
)

type recordingTx struct {
	pgx.Tx
	sql  string
	args []any
	tag  pgconn.CommandTag
	err  error
}

func (r *recordingTx) Exec(_ context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	r.sql, r.args = sql, args
	return r.tag, r.err
}

func inTx(tx *recordingTx) context.Context {
	return context.WithValue(context.Background(), txKey{}, pgx.Tx(tx))
}

func newView(t *testing.T) *lotview.LotView {
	t.Helper()
	v, err := lotview.New(lotview.NewParams{
		LotID:        uuid.New(),
		Code:         "LT-001",
		Variety:      "Castillo",
		Process:      lotview.ProcessWashed,
		HarvestDate:  time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC),
		FarmID:       uuid.New(),
		FarmName:     "La Esperanza",
		FarmAltitude: 1800,
		ProducerID:   uuid.New(),
		ProducerName: "Maria Lopez",
		CustodianID:  uuid.New(),
		CurrentStage: lotview.StageNone,
		Status:       lotview.StatusActive,
		Version:      2,
		CreatedAt:    time.Now().Add(-time.Hour),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return v
}

func assertInternal(t *testing.T, err error) {
	t.Helper()
	ex, ok := exception.From(err)
	if !ok || ex.Kind() != exception.KindInternal {
		t.Errorf("got %v, want an internal error", err)
	}
}

func TestLotViewRepositorySaveUpsertsGuardedByVersion(t *testing.T) {
	tx := &recordingTx{}
	view := newView(t)

	if err := NewLotViewRepository(nil).Save(inTx(tx), view); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if tx.sql != upsertLotView {
		t.Fatal("expected the lot view upsert")
	}
	if len(tx.args) != 17 || tx.args[0] != view.LotID() || tx.args[14] != 2 {
		t.Errorf("unexpected args %v", tx.args)
	}
}

func TestLotViewRepositorySaveWrapsDatabaseErrors(t *testing.T) {
	tx := &recordingTx{err: errors.New("boom")}

	assertInternal(t, NewLotViewRepository(nil).Save(inTx(tx), newView(t)))
}

func TestLotViewRepositorySaveRejectsDefault(t *testing.T) {
	tx := &recordingTx{}

	assertInternal(t, NewLotViewRepository(nil).Save(inTx(tx), lotview.Default()))
	if tx.sql != "" {
		t.Error("nothing must be written for a default view")
	}
}

func TestQRCodeRepositorySaveInsertsIgnoringConflicts(t *testing.T) {
	tx := &recordingTx{}
	code, err := qrcode.New(uuid.New(), "https://example.com", qrcode.DefaultErrorCorrection, "v1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := NewQRCodeRepository(nil).Save(inTx(tx), code); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if tx.sql != insertQRCode || len(tx.args) != 5 || tx.args[0] != code.LotID() || tx.args[2] != "Q" {
		t.Errorf("unexpected insert %q %v", tx.sql, tx.args)
	}
}

func TestQRCodeRepositorySaveRejectsDefault(t *testing.T) {
	tx := &recordingTx{}

	assertInternal(t, NewQRCodeRepository(nil).Save(inTx(tx), qrcode.Default()))
	if tx.sql != "" {
		t.Error("nothing must be written for a default code")
	}
}

func TestScanRepositorySaveInsertsScan(t *testing.T) {
	tx := &recordingTx{}
	s, err := scan.New(uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := NewScanRepository(nil).Save(inTx(tx), s); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if tx.sql != insertScan || len(tx.args) != 2 || tx.args[0] != s.LotID() {
		t.Errorf("unexpected insert %q %v", tx.sql, tx.args)
	}
}

func TestScanRepositorySaveRejectsDefaultAndWrapsErrors(t *testing.T) {
	assertInternal(t, NewScanRepository(nil).Save(inTx(&recordingTx{}), scan.Default()))

	s, _ := scan.New(uuid.New())
	assertInternal(t, NewScanRepository(nil).Save(inTx(&recordingTx{err: errors.New("boom")}), s))
}

func TestProcessedEventRepositoryReportsFirstAndDuplicate(t *testing.T) {
	id := uuid.New()

	fresh := &recordingTx{tag: pgconn.NewCommandTag("INSERT 0 1")}
	ok, err := NewProcessedEventRepository(nil).Register(inTx(fresh), id, "lot.created")
	if err != nil || !ok {
		t.Fatalf("expected first registration, got %v %v", ok, err)
	}
	if fresh.sql != insertProcessedEvent || fresh.args[0] != id || fresh.args[1] != "lot.created" {
		t.Errorf("unexpected insert %q %v", fresh.sql, fresh.args)
	}

	duplicate := &recordingTx{tag: pgconn.NewCommandTag("INSERT 0 0")}
	ok, err = NewProcessedEventRepository(nil).Register(inTx(duplicate), id, "lot.created")
	if err != nil || ok {
		t.Fatalf("expected duplicate to be reported, got %v %v", ok, err)
	}
}

func TestProcessedEventRepositoryWrapsErrors(t *testing.T) {
	tx := &recordingTx{err: errors.New("boom")}

	ok, err := NewProcessedEventRepository(nil).Register(inTx(tx), uuid.New(), "lot.created")

	if ok {
		t.Error("expected false on error")
	}
	assertInternal(t, err)
}
