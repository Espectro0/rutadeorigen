package helper_test

import (
	"testing"

	"rutadeorigen/traceability/internal/platform/helper"
)

func TestIsNil(t *testing.T) {
	value := 10

	if !helper.IsNil[int](nil) {
		t.Error("expected nil pointer to be reported as nil")
	}
	if helper.IsNil(&value) {
		t.Error("expected non-nil pointer to be reported as not nil")
	}
}

func TestValueOrDefault(t *testing.T) {
	t.Run("nil pointer returns default", func(t *testing.T) {
		if got := helper.ValueOrDefault(nil, "default"); got != "default" {
			t.Errorf("got %q, want %q", got, "default")
		}
	})

	t.Run("non-nil pointer returns its value", func(t *testing.T) {
		value := "Castillo"
		if got := helper.ValueOrDefault(&value, "default"); got != "Castillo" {
			t.Errorf("got %q, want %q", got, "Castillo")
		}
	})

	t.Run("zero value is returned as is, not replaced", func(t *testing.T) {
		value := 0
		if got := helper.ValueOrDefault(&value, 99); got != 0 {
			t.Errorf("got %d, want 0", got)
		}
	})
}
