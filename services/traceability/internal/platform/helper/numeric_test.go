package helper_test

import (
	"testing"

	"rutadeorigen/traceability/internal/platform/helper"
)

func TestDefaultInt(t *testing.T) {
	if got := helper.DefaultInt(); got != 0 {
		t.Errorf("got %d, want 0", got)
	}
}

func TestIntOrDefault(t *testing.T) {
	value := 1650

	if got := helper.IntOrDefault(nil); got != helper.DefaultInt() {
		t.Errorf("nil: got %d, want default", got)
	}
	if got := helper.IntOrDefault(&value); got != 1650 {
		t.Errorf("got %d, want 1650", got)
	}
}

func TestIsDefaultInt(t *testing.T) {
	if !helper.IsDefaultInt(0) {
		t.Error("expected 0 to be the default int")
	}
	if helper.IsDefaultInt(-1) {
		t.Error("expected -1 not to be the default int")
	}
}

func TestDefaultFloat(t *testing.T) {
	if got := helper.DefaultFloat(); got != 0.0 {
		t.Errorf("got %v, want 0.0", got)
	}
}

func TestFloatOrDefault(t *testing.T) {
	value := 70.5

	if got := helper.FloatOrDefault(nil); got != helper.DefaultFloat() {
		t.Errorf("nil: got %v, want default", got)
	}
	if got := helper.FloatOrDefault(&value); got != 70.5 {
		t.Errorf("got %v, want 70.5", got)
	}
}

func TestIsDefaultFloat(t *testing.T) {
	if !helper.IsDefaultFloat(0.0) {
		t.Error("expected 0.0 to be the default float")
	}
	if helper.IsDefaultFloat(0.1) {
		t.Error("expected 0.1 not to be the default float")
	}
}

func TestIsInRangeInt(t *testing.T) {
	tests := []struct {
		name  string
		value int
		want  bool
	}{
		{"exactly min", 0, true},
		{"exactly max", 3000, true},
		{"inside range", 1650, true},
		{"just below min", -1, false},
		{"just above max", 3001, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := helper.IsInRange(tt.value, 0, 3000); got != tt.want {
				t.Errorf("IsInRange(%d, 0, 3000) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

func TestIsInRangeFloat(t *testing.T) {
	if !helper.IsInRange(0.5, 0.0, 1.0) {
		t.Error("expected 0.5 to be in range [0, 1]")
	}
	if helper.IsInRange(1.01, 0.0, 1.0) {
		t.Error("expected 1.01 not to be in range [0, 1]")
	}
}
