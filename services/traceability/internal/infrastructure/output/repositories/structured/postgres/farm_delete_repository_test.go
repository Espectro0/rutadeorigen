package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"rutadeorigen/traceability/internal/platform/exception"
)

func runDeleteFarm(tx *stageTx, id uuid.UUID) error {
	ctx := context.WithValue(context.Background(), txKey{}, pgx.Tx(tx))
	return NewFarmRepository(nil).Delete(ctx, id)
}

func TestFarmRepositoryDeleteRemovesTheFarmInTheCurrentTransaction(t *testing.T) {
	tx := &stageTx{rowsAffected: 1}
	id := uuid.New()

	if err := runDeleteFarm(tx, id); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if tx.execSQL != deleteFarm || len(tx.execArgs) != 1 || tx.execArgs[0] != id {
		t.Errorf("unexpected statement %q with args %v", tx.execSQL, tx.execArgs)
	}
}

func TestFarmRepositoryDeleteReportsAMissingFarm(t *testing.T) {
	err := runDeleteFarm(&stageTx{rowsAffected: 0}, uuid.New())

	if !exception.HasCode(err, exception.CodeFarmNotFound) {
		t.Errorf("got %v, want code %q", err, exception.CodeFarmNotFound)
	}
}

func TestFarmRepositoryDeleteReportsAFarmWithLots(t *testing.T) {
	err := runDeleteFarm(&stageTx{execErr: &pgconn.PgError{Code: foreignKeyViolation}}, uuid.New())

	if !exception.HasCode(err, exception.CodeFarmHasLots) {
		t.Errorf("got %v, want code %q", err, exception.CodeFarmHasLots)
	}
}

func TestFarmRepositoryDeleteFailsOnOtherErrors(t *testing.T) {
	cause := errors.New("connection reset")

	err := runDeleteFarm(&stageTx{execErr: cause}, uuid.New())

	if !exception.HasCode(err, exception.CodeInternal) || !errors.Is(err, cause) {
		t.Errorf("got %v, want an internal error wrapping the cause", err)
	}
}

func TestIsForeignKeyViolation(t *testing.T) {
	if !isForeignKeyViolation(&pgconn.PgError{Code: foreignKeyViolation}) {
		t.Error("expected a foreign key violation")
	}
	if isForeignKeyViolation(&pgconn.PgError{Code: uniqueViolation}) || isForeignKeyViolation(errors.New("other")) || isForeignKeyViolation(nil) {
		t.Error("only foreign key violations must match")
	}
}
