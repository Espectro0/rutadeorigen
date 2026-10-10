package helper_test

import (
	"testing"

	"github.com/google/uuid"

	"rutadeorigen/traceability/internal/platform/helper"
)

func TestDefaultUUID(t *testing.T) {
	if got := helper.DefaultUUID(); got != uuid.Nil {
		t.Errorf("got %s, want %s", got, uuid.Nil)
	}
}

func TestUUIDOrDefault(t *testing.T) {
	value := uuid.New()

	if got := helper.UUIDOrDefault(nil); got != helper.DefaultUUID() {
		t.Errorf("nil: got %s, want default UUID", got)
	}
	if got := helper.UUIDOrDefault(&value); got != value {
		t.Errorf("got %s, want %s", got, value)
	}
}

func TestIsDefaultUUID(t *testing.T) {
	if !helper.IsDefaultUUID(uuid.Nil) {
		t.Error("expected uuid.Nil to be the default UUID")
	}
	if helper.IsDefaultUUID(uuid.New()) {
		t.Error("expected a new UUID not to be the default UUID")
	}
}
