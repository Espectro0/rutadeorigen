package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"

	"rutadeorigen/traceability/internal/domain/lot"
	"rutadeorigen/traceability/internal/platform/exception"
)

func voidedLot(t *testing.T) *lot.Lot {
	t.Helper()

	l := advancedLot(t)
	if err := l.Void("Registered by mistake"); err != nil {
		t.Fatal(err)
	}
	return l
}

func updateStatus(tx *stageTx, l *lot.Lot, expectedVersion int) error {
	ctx := context.WithValue(context.Background(), txKey{}, pgx.Tx(tx))
	return NewLotRepository(nil).UpdateStatus(ctx, l, expectedVersion)
}

func TestLotRepositoryUpdateStatusGuardsTheVersionInTheCurrentTransaction(t *testing.T) {
	tx := &stageTx{rowsAffected: 1}
	l := voidedLot(t)

	if err := updateStatus(tx, l, 2); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if tx.execSQL != updateLotStatus {
		t.Fatal("expected the lot status update")
	}
	want := []any{l.ID(), "voided", "Registered by mistake", 3, 2}
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

func TestLotRepositoryUpdateStatusReportsAVersionConflict(t *testing.T) {
	tx := &stageTx{rowsAffected: 0, exists: true}
	l := voidedLot(t)

	err := updateStatus(tx, l, 2)

	if !exception.HasCode(err, exception.CodeLotVersionConflict) {
		t.Errorf("got %v, want code %q", err, exception.CodeLotVersionConflict)
	}
	if tx.querySQL != existsLotByID || len(tx.queryArgs) != 1 || tx.queryArgs[0] != l.ID() {
		t.Error("expected the existence check for the lot")
	}
}

func TestLotRepositoryUpdateStatusReportsAMissingLot(t *testing.T) {
	err := updateStatus(&stageTx{rowsAffected: 0, exists: false}, voidedLot(t), 2)

	if !exception.HasCode(err, exception.CodeLotNotFound) {
		t.Errorf("got %v, want code %q", err, exception.CodeLotNotFound)
	}
}

func TestLotRepositoryUpdateStatusRejectsADefaultLot(t *testing.T) {
	tx := &stageTx{rowsAffected: 1}

	err := updateStatus(tx, lot.Default(), 1)

	if !exception.HasCode(err, exception.CodeInternal) {
		t.Errorf("got %v, want code %q", err, exception.CodeInternal)
	}
	if tx.execSQL != "" {
		t.Error("a default lot must never reach the database")
	}
}

func TestLotRepositoryUpdateStatusFailures(t *testing.T) {
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
			err := updateStatus(tt.tx, voidedLot(t), 2)

			if !exception.HasCode(err, exception.CodeInternal) {
				t.Errorf("got %v, want code %q", err, exception.CodeInternal)
			}
			if !errors.Is(err, cause) {
				t.Error("expected the original cause to be kept")
			}
		})
	}
}
