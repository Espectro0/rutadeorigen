package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"rutadeorigen/traceability/internal/domain/farm"
	"rutadeorigen/traceability/internal/platform/exception"
)

func saveFarm(t *testing.T, tx *recordingTx, f *farm.Farm) error {
	t.Helper()
	ctx := context.WithValue(context.Background(), txKey{}, pgx.Tx(tx))
	return NewFarmRepository(nil).Save(ctx, f)
}

func TestFarmRepositorySaveInsertsTheFarmInTheCurrentTransaction(t *testing.T) {
	tx := &recordingTx{}
	createdAt := time.Date(2026, time.October, 21, 15, 0, 0, 0, time.UTC)
	f := farm.Restore(uuid.New(), uuid.New(), "La Esperanza", 1650, createdAt)

	if err := saveFarm(t, tx, f); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if tx.sql != insertFarm {
		t.Fatal("expected the farm insert")
	}
	want := []any{f.ID(), f.ProducerID(), "La Esperanza", 1650, createdAt}
	if len(tx.args) != len(want) {
		t.Fatalf("got %d args, want %d", len(tx.args), len(want))
	}
	for i := range want {
		if tx.args[i] != want[i] {
			t.Errorf("arg %d: got %v, want %v", i, tx.args[i], want[i])
		}
	}
}

func TestFarmRepositorySaveRejectsADefaultFarm(t *testing.T) {
	for _, f := range []*farm.Farm{farm.Default(), nil} {
		tx := &recordingTx{}

		err := saveFarm(t, tx, f)

		e, ok := exception.From(err)
		if !ok || e.Kind() != exception.KindInternal {
			t.Errorf("got %v, want an internal error", err)
		}
		if tx.sql != "" {
			t.Error("nothing must be inserted for a default farm")
		}
	}
}

func TestFarmRepositorySaveFailsWhenTheInsertFails(t *testing.T) {
	cause := errors.New("connection reset")
	tx := &recordingTx{err: cause}

	err := saveFarm(t, tx, farm.Restore(uuid.New(), uuid.New(), "La Esperanza", 1650, time.Now().UTC()))

	e, ok := exception.From(err)
	if !ok || e.Kind() != exception.KindInternal {
		t.Errorf("got %v, want an internal error", err)
	}
	if !errors.Is(err, cause) {
		t.Error("expected the original cause to be kept")
	}
}
