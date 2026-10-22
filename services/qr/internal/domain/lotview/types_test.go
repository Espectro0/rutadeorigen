package lotview_test

import (
	"testing"

	"rutadeorigen/qr/internal/domain/lotview"
)

func TestParseProcess(t *testing.T) {
	tests := []struct {
		value string
		want  lotview.Process
	}{
		{"washed", lotview.ProcessWashed},
		{"NATURAL", lotview.ProcessNatural},
		{"  honey  ", lotview.ProcessHoney},
		{"", lotview.ProcessUnknown},
		{"roasted", lotview.ProcessUnknown},
		{"unknown", lotview.ProcessUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			if got := lotview.ParseProcess(tt.value); got != tt.want {
				t.Errorf("ParseProcess(%q) = %q, want %q", tt.value, got, tt.want)
			}
		})
	}
}

func TestParseStatus(t *testing.T) {
	if lotview.ParseStatus("ACTIVE") != lotview.StatusActive {
		t.Error("expected active")
	}
	if lotview.ParseStatus(" voided ") != lotview.StatusVoided {
		t.Error("expected voided")
	}
	if lotview.ParseStatus("") != lotview.StatusUnknown {
		t.Error("expected unknown")
	}
}

func TestParseStage(t *testing.T) {
	tests := []struct {
		value string
		want  lotview.Stage
	}{
		{"none", lotview.StageNone},
		{"FARM", lotview.StageFarm},
		{" roasting ", lotview.StageRoasting},
		{"packaging", lotview.StagePackaging},
		{"", lotview.StageUnknown},
		{"shipping", lotview.StageUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			if got := lotview.ParseStage(tt.value); got != tt.want {
				t.Errorf("ParseStage(%q) = %q, want %q", tt.value, got, tt.want)
			}
		})
	}
}

func TestIsValidAndString(t *testing.T) {
	if !lotview.ProcessHoney.IsValid() || lotview.ProcessUnknown.IsValid() {
		t.Error("Process.IsValid failed")
	}
	if !lotview.StatusVoided.IsValid() || lotview.StatusUnknown.IsValid() {
		t.Error("Status.IsValid failed")
	}
	if !lotview.StageNone.IsValid() || lotview.StageUnknown.IsValid() {
		t.Error("Stage.IsValid failed")
	}
	if lotview.ProcessWashed.String() != "washed" || lotview.StatusActive.String() != "active" || lotview.StagePackaging.String() != "packaging" {
		t.Error("String failed")
	}
}
