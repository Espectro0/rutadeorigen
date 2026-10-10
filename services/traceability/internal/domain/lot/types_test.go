package lot_test

import (
	"testing"

	"rutadeorigen/traceability/internal/domain/lot"
)

func TestParseProcess(t *testing.T) {
	tests := []struct {
		value string
		want  lot.Process
	}{
		{"washed", lot.ProcessWashed},
		{"NATURAL", lot.ProcessNatural},
		{"  honey  ", lot.ProcessHoney},
		{"", lot.ProcessUnknown},
		{"roasted", lot.ProcessUnknown},
		{"unknown", lot.ProcessUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			if got := lot.ParseProcess(tt.value); got != tt.want {
				t.Errorf("ParseProcess(%q) = %q, want %q", tt.value, got, tt.want)
			}
		})
	}
}

func TestParseStatus(t *testing.T) {
	if lot.ParseStatus("ACTIVE") != lot.StatusActive {
		t.Error("expected active")
	}
	if lot.ParseStatus(" voided ") != lot.StatusVoided {
		t.Error("expected voided")
	}
	if lot.ParseStatus("") != lot.StatusUnknown {
		t.Error("expected unknown")
	}
}

func TestParseEventType(t *testing.T) {
	tests := []struct {
		value string
		want  lot.EventType
	}{
		{"stage", lot.EventTypeStage},
		{"Transfer", lot.EventTypeTransfer},
		{" correction ", lot.EventTypeCorrection},
		{"delete", lot.EventTypeUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			if got := lot.ParseEventType(tt.value); got != tt.want {
				t.Errorf("ParseEventType(%q) = %q, want %q", tt.value, got, tt.want)
			}
		})
	}
}

func TestIsValidAndString(t *testing.T) {
	if !lot.ProcessHoney.IsValid() || lot.ProcessUnknown.IsValid() {
		t.Error("Process.IsValid failed")
	}
	if !lot.StatusVoided.IsValid() || !lot.EventTypeStage.IsValid() {
		t.Error("IsValid failed for a valid value")
	}
	if lot.ProcessWashed.String() != "washed" || lot.StageNone.String() != "none" {
		t.Error("String failed")
	}
	if lot.StatusActive.String() != "active" || lot.EventTypeTransfer.String() != "transfer" {
		t.Error("String failed")
	}
}
