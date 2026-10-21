package memory_test

import (
	"context"
	"errors"
	"testing"

	"rutadeorigen/traceability/internal/infrastructure/output/repositories/structured/memory"
)

type ctxKey struct{}

func TestTxManagerRunsTheFunctionWithTheSameContext(t *testing.T) {
	ctx := context.WithValue(context.Background(), ctxKey{}, "value")
	calls := 0

	err := memory.NewTxManager().WithinTx(ctx, func(got context.Context) error {
		calls++
		if got.Value(ctxKey{}) != "value" {
			t.Error("expected the original context")
		}
		return nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 1 {
		t.Errorf("got %d calls, want 1", calls)
	}
}

func TestTxManagerReturnsTheFunctionError(t *testing.T) {
	want := errors.New("boom")

	err := memory.NewTxManager().WithinTx(context.Background(), func(context.Context) error {
		return want
	})

	if !errors.Is(err, want) {
		t.Errorf("got %v, want %v", err, want)
	}
}
