package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"rutadeorigen/traceability/internal/domain/lot"
	"rutadeorigen/traceability/internal/platform/exception"
)

type stageTx struct {
	pgx.Tx
	rowsAffected int64
	execErr      error
	exists       bool
	queryErr     error
	execSQL      string
	execArgs     []any
	querySQL     string
	queryArgs    []any
}

func (s *stageTx) Exec(_ context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	s.execSQL, s.execArgs = sql, args
	if s.execErr != nil {
		return pgconn.CommandTag{}, s.execErr
	}
	if s.rowsAffected == 0 {
		return pgconn.NewCommandTag("UPDATE 0"), nil
	}
	return pgconn.NewCommandTag("UPDATE 1"), nil
}

func (s *stageTx) QueryRow(_ context.Context, sql string, args ...any) pgx.Row {
	s.querySQL, s.queryArgs = sql, args
	return stageRow{exists: s.exists, err: s.queryErr}
}

type stageRow struct {
	exists bool
	err    error
}

func (r stageRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	*(dest[0].(*bool)) = r.exists
	return nil
}

func advancedLot(t *testing.T) *lot.Lot {
	t.Helper()
	l, err := lot.New(uuid.New(), uuid.New(), "LT-001", "Castillo", lot.ProcessWashed, time.Now().AddDate(0, -1, 0))
	if err != nil {
		t.Fatal(err)
	}
	if err := l.AdvanceTo(lot.StageFarm); err != nil {
		t.Fatal(err)
	}
	return l
}

func updateStage(tx *stageTx, l *lot.Lot, expectedVersion int) error {
	ctx := context.WithValue(context.Background(), txKey{}, pgx.Tx(tx))
	return NewLotRepository(nil).UpdateStage(ctx, l, expectedVersion)
}

func TestLotRepositoryUpdateStageGuardsTheVersionInTheCurrentTransaction(t *testing.T) {
	tx := &stageTx{rowsAffected: 1}
	l := advancedLot(t)

	if err := updateStage(tx, l, 1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if tx.execSQL != updateLotStage {
		t.Fatal("expected the lot stage update")
	}
	want := []any{l.ID(), "farm", 2, 1}
	if len(tx.execArgs) != len(want) {
		t.Fatalf("got %d args, want %d", len(tx.execArgs), len(want))
	}
	for i := range want {
		if tx.execArgs[i] != want[i] {
			t.Errorf("arg %d: got %v, want %v", i, tx.execArgs[i], want[i])
		}
	}
	if tx.querySQL != "" {
		t.Error("the existence check must only run when no row was updated")
	}
}

func TestLotRepositoryUpdateStageReportsAVersionConflict(t *testing.T) {
	tx := &stageTx{rowsAffected: 0, exists: true}
	l := advancedLot(t)

	err := updateStage(tx, l, 1)

	if !exception.HasCode(err, exception.CodeLotVersionConflict) {
		t.Errorf("got %v, want code %q", err, exception.CodeLotVersionConflict)
	}
	if tx.querySQL != existsLotByID || len(tx.queryArgs) != 1 || tx.queryArgs[0] != l.ID() {
		t.Error("expected the existence check for the lot")
	}
}

func TestLotRepositoryUpdateStageReportsAMissingLot(t *testing.T) {
	tx := &stageTx{rowsAffected: 0, exists: false}

	err := updateStage(tx, advancedLot(t), 1)

	if !exception.HasCode(err, exception.CodeLotNotFound) {
		t.Errorf("got %v, want code %q", err, exception.CodeLotNotFound)
	}
}

func TestLotRepositoryUpdateStageRejectsADefaultLot(t *testing.T) {
	for _, l := range []*lot.Lot{lot.Default(), nil} {
		tx := &stageTx{rowsAffected: 1}

		err := updateStage(tx, l, 1)

		e, ok := exception.From(err)
		if !ok || e.Kind() != exception.KindInternal {
			t.Errorf("got %v, want an internal error", err)
		}
		if tx.execSQL != "" {
			t.Error("nothing must be updated for a default lot")
		}
	}
}

func TestLotRepositoryUpdateStageFailures(t *testing.T) {
	cause := errors.New("connection reset")

	tests := []struct {
		name string
		tx   *stageTx
	}{
		{"the update fails", &stageTx{execErr: cause}},
		{"the existence check fails", &stageTx{rowsAffected: 0, queryErr: cause}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := updateStage(tt.tx, advancedLot(t), 1)

			e, ok := exception.From(err)
			if !ok || e.Kind() != exception.KindInternal {
				t.Errorf("got %v, want an internal error", err)
			}
			if !errors.Is(err, cause) {
				t.Error("expected the original cause to be kept")
			}
		})
	}
}
