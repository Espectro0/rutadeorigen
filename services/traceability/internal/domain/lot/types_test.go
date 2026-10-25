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

func TestParseDryingMethod(t *testing.T) {
	tests := []struct {
		value string
		want  lot.DryingMethod
	}{
		{"sun_patio", lot.DryingMethodSunPatio},
		{"AFRICAN_BED", lot.DryingMethodAfricanBed},
		{"  greenhouse  ", lot.DryingMethodGreenhouse},
		{"Mechanical", lot.DryingMethodMechanical},
		{"", lot.DryingMethodUnknown},
		{"unknown", lot.DryingMethodUnknown},
		{"oven", lot.DryingMethodUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			if got := lot.ParseDryingMethod(tt.value); got != tt.want {
				t.Errorf("ParseDryingMethod(%q) = %q, want %q", tt.value, got, tt.want)
			}
		})
	}
}

func TestParseRoastLevel(t *testing.T) {
	tests := []struct {
		value string
		want  lot.RoastLevel
	}{
		{"light", lot.RoastLevelLight},
		{"MEDIUM", lot.RoastLevelMedium},
		{"  dark  ", lot.RoastLevelDark},
		{"", lot.RoastLevelUnknown},
		{"unknown", lot.RoastLevelUnknown},
		{"burnt", lot.RoastLevelUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			if got := lot.ParseRoastLevel(tt.value); got != tt.want {
				t.Errorf("ParseRoastLevel(%q) = %q, want %q", tt.value, got, tt.want)
			}
		})
	}
}

func TestParseGrind(t *testing.T) {
	tests := []struct {
		value string
		want  lot.Grind
	}{
		{"whole_bean", lot.GrindWholeBean},
		{"GROUND", lot.GrindGround},
		{"  ground  ", lot.GrindGround},
		{"", lot.GrindUnknown},
		{"unknown", lot.GrindUnknown},
		{"powder", lot.GrindUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			if got := lot.ParseGrind(tt.value); got != tt.want {
				t.Errorf("ParseGrind(%q) = %q, want %q", tt.value, got, tt.want)
			}
		})
	}
}

func TestStageDetailsEnumsIsValidAndString(t *testing.T) {
	if !lot.DryingMethodAfricanBed.IsValid() || lot.DryingMethodUnknown.IsValid() || lot.DryingMethod("oven").IsValid() {
		t.Error("DryingMethod.IsValid failed")
	}
	if !lot.RoastLevelDark.IsValid() || lot.RoastLevelUnknown.IsValid() || lot.RoastLevel("burnt").IsValid() {
		t.Error("RoastLevel.IsValid failed")
	}
	if !lot.GrindWholeBean.IsValid() || lot.GrindUnknown.IsValid() || lot.Grind("powder").IsValid() {
		t.Error("Grind.IsValid failed")
	}
	if lot.DryingMethodSunPatio.String() != "sun_patio" || lot.DryingMethodAfricanBed.String() != "african_bed" {
		t.Error("DryingMethod.String failed")
	}
	if lot.RoastLevelMedium.String() != "medium" || lot.GrindGround.String() != "ground" {
		t.Error("String failed")
	}
}
