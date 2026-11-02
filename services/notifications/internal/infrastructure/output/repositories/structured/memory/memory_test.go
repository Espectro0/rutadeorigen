package memory

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
)

func TestProcessedEventRepository(t *testing.T) {
	repository := NewProcessedEventRepository()
	messageID := uuid.New()

	if found, _ := repository.Exists(context.Background(), messageID); found {
		t.Error("unknown message must not exist")
	}
	if first, _ := repository.Register(context.Background(), messageID, "evidence.processed"); !first {
		t.Error("first registration must report true")
	}
	if first, _ := repository.Register(context.Background(), messageID, "evidence.processed"); first {
		t.Error("second registration must report false")
	}
	if found, _ := repository.Exists(context.Background(), messageID); !found {
		t.Error("registered message must exist")
	}
}

func TestTxManagerRunsFunctionAndReturnsItsError(t *testing.T) {
	failure := errors.New("boom")
	ran := false

	err := NewTxManager().WithinTx(context.Background(), func(context.Context) error {
		ran = true
		return failure
	})

	if !ran || !errors.Is(err, failure) {
		t.Errorf("ran=%v err=%v", ran, err)
	}
}
