package actor_test

import (
	"testing"

	"rutadeorigen/traceability/internal/domain/actor"
)

func TestParseType(t *testing.T) {
	tests := []struct {
		value string
		want  actor.Type
	}{
		{"organization", actor.TypeOrganization},
		{"Producer", actor.TypeProducer},
		{" roaster ", actor.TypeRoaster},
		{"BRAND", actor.TypeBrand},
		{"", actor.TypeUnknown},
		{"consumer", actor.TypeUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			if got := actor.ParseType(tt.value); got != tt.want {
				t.Errorf("ParseType(%q) = %q, want %q", tt.value, got, tt.want)
			}
		})
	}
}

func TestParseStatus(t *testing.T) {
	if actor.ParseStatus(" Active ") != actor.StatusActive {
		t.Error("expected active")
	}
	if actor.ParseStatus("inactive") != actor.StatusInactive {
		t.Error("expected inactive")
	}
	if actor.ParseStatus("deleted") != actor.StatusUnknown {
		t.Error("expected unknown")
	}
}

func TestIsValidAndString(t *testing.T) {
	if !actor.TypeBrand.IsValid() || actor.Type("admin").IsValid() {
		t.Error("Type.IsValid failed")
	}
	if !actor.StatusActive.IsValid() || actor.StatusUnknown.IsValid() {
		t.Error("Status.IsValid failed")
	}
	if actor.TypeRoaster.String() != "roaster" || actor.StatusInactive.String() != "inactive" {
		t.Error("String failed")
	}
}
