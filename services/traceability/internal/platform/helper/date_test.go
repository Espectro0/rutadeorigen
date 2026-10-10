package helper_test

import (
	"testing"
	"time"

	"rutadeorigen/traceability/internal/platform/helper"
)

func TestDefaultDate(t *testing.T) {
	if got := helper.DefaultDate(); !got.IsZero() {
		t.Errorf("got %v, want zero time", got)
	}
}

func TestDateOrDefault(t *testing.T) {
	value := time.Date(2026, time.October, 9, 0, 0, 0, 0, time.UTC)

	if got := helper.DateOrDefault(nil); !helper.IsDefaultDate(got) {
		t.Errorf("nil: got %v, want default date", got)
	}
	if got := helper.DateOrDefault(&value); !got.Equal(value) {
		t.Errorf("got %v, want %v", got, value)
	}
}

func TestIsDefaultDate(t *testing.T) {
	if !helper.IsDefaultDate(time.Time{}) {
		t.Error("expected zero time to be the default date")
	}
	if helper.IsDefaultDate(time.Now()) {
		t.Error("expected current time not to be the default date")
	}
}
