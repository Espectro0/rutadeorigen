package lot_test

import (
	"testing"

	"rutadeorigen/traceability/internal/domain/lot"
)

func TestParseStage(t *testing.T) {
	tests := []struct {
		value string
		want  lot.Stage
	}{
		{"none", lot.StageNone},
		{"farm", lot.StageFarm},
		{"ROASTING", lot.StageRoasting},
		{"  packaging  ", lot.StagePackaging},
		{"", lot.StageUnknown},
		{"harvest", lot.StageUnknown},
		{"unknown", lot.StageUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			if got := lot.ParseStage(tt.value); got != tt.want {
				t.Errorf("ParseStage(%q) = %q, want %q", tt.value, got, tt.want)
			}
		})
	}
}

func TestStageIsValid(t *testing.T) {
	for _, s := range []lot.Stage{lot.StageNone, lot.StageFarm, lot.StageRoasting, lot.StagePackaging} {
		if !s.IsValid() {
			t.Errorf("expected %q to be valid", s)
		}
	}
	if lot.StageUnknown.IsValid() || lot.Stage("drying").IsValid() {
		t.Error("expected unknown stages to be invalid")
	}
}

func TestStageCanFollow(t *testing.T) {
	tests := []struct {
		name    string
		next    lot.Stage
		current lot.Stage
		want    bool
	}{
		{"none to farm", lot.StageFarm, lot.StageNone, true},
		{"farm to roasting", lot.StageRoasting, lot.StageFarm, true},
		{"roasting to packaging", lot.StagePackaging, lot.StageRoasting, true},

		{"skip roasting", lot.StagePackaging, lot.StageFarm, false},
		{"skip farm", lot.StageRoasting, lot.StageNone, false},
		{"repeat farm", lot.StageFarm, lot.StageFarm, false},
		{"go back to farm", lot.StageFarm, lot.StageRoasting, false},
		{"nothing after packaging", lot.StagePackaging, lot.StagePackaging, false},
		{"none never follows", lot.StageNone, lot.StageNone, false},
		{"unknown next", lot.StageUnknown, lot.StageNone, false},
		{"unknown current", lot.StageFarm, lot.StageUnknown, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.next.CanFollow(tt.current); got != tt.want {
				t.Errorf("%q.CanFollow(%q) = %v, want %v", tt.next, tt.current, got, tt.want)
			}
		})
	}
}

func TestStageNext(t *testing.T) {
	tests := []struct {
		stage lot.Stage
		want  lot.Stage
	}{
		{lot.StageNone, lot.StageFarm},
		{lot.StageFarm, lot.StageRoasting},
		{lot.StageRoasting, lot.StagePackaging},
		{lot.StagePackaging, lot.StageUnknown},
		{lot.StageUnknown, lot.StageUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.stage.String(), func(t *testing.T) {
			if got := tt.stage.Next(); got != tt.want {
				t.Errorf("%q.Next() = %q, want %q", tt.stage, got, tt.want)
			}
		})
	}
}
