package lot_test

import (
	"testing"
	"time"

	"rutadeorigen/traceability/internal/domain/lot"
	"rutadeorigen/traceability/internal/platform/exception"
	"rutadeorigen/traceability/internal/platform/helper"
)

func stageDetailsEventDate() time.Time {
	return helper.DateUTC(time.Now().AddDate(0, 0, -1))
}

func validFarmDetails() lot.FarmDetails {
	return lot.FarmDetails{
		DryingMethod:      lot.DryingMethodSunPatio,
		DryingDays:        15,
		MoisturePercent:   11.5,
		ParchmentWeightKg: 120.5,
	}
}

func validRoastingDetails() lot.RoastingDetails {
	return lot.RoastingDetails{
		RoastLevel:      lot.RoastLevelMedium,
		DurationMinutes: 14,
		GreenWeightKg:   100,
		RoastedWeightKg: 84,
	}
}

func validPackagingDetails() lot.PackagingDetails {
	return lot.PackagingDetails{
		Grind:          lot.GrindWholeBean,
		PackageWeightG: 250,
		Units:          40,
		BestBefore:     stageDetailsEventDate().AddDate(0, 6, 0).Format(time.DateOnly),
	}
}

func TestStageDetailsImplementInterface(t *testing.T) {
	tests := []struct {
		name    string
		details lot.StageDetails
		want    lot.Stage
	}{
		{"farm", validFarmDetails(), lot.StageFarm},
		{"roasting", validRoastingDetails(), lot.StageRoasting},
		{"packaging", validPackagingDetails(), lot.StagePackaging},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.details.Stage() != tt.want {
				t.Errorf("Stage() = %q, want %q", tt.details.Stage(), tt.want)
			}
			if err := tt.details.Validate(stageDetailsEventDate()); err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestFarmDetailsValidate(t *testing.T) {
	tests := []struct {
		name     string
		mutate   func(*lot.FarmDetails)
		wantCode exception.Code
	}{
		{"valid", func(*lot.FarmDetails) {}, ""},
		{"african bed", func(d *lot.FarmDetails) { d.DryingMethod = lot.DryingMethodAfricanBed }, ""},
		{"unknown drying method", func(d *lot.FarmDetails) { d.DryingMethod = lot.DryingMethodUnknown }, exception.CodeStageInvalidDryingMethod},
		{"empty drying method", func(d *lot.FarmDetails) { d.DryingMethod = "" }, exception.CodeStageInvalidDryingMethod},
		{"invalid drying method", func(d *lot.FarmDetails) { d.DryingMethod = "oven" }, exception.CodeStageInvalidDryingMethod},
		{"drying days below minimum", func(d *lot.FarmDetails) { d.DryingDays = 0 }, exception.CodeStageInvalidDryingDays},
		{"negative drying days", func(d *lot.FarmDetails) { d.DryingDays = -3 }, exception.CodeStageInvalidDryingDays},
		{"drying days at minimum", func(d *lot.FarmDetails) { d.DryingDays = 1 }, ""},
		{"drying days at maximum", func(d *lot.FarmDetails) { d.DryingDays = 60 }, ""},
		{"drying days above maximum", func(d *lot.FarmDetails) { d.DryingDays = 61 }, exception.CodeStageInvalidDryingDays},
		{"moisture below minimum", func(d *lot.FarmDetails) { d.MoisturePercent = 7.99 }, exception.CodeStageInvalidMoisture},
		{"moisture at minimum", func(d *lot.FarmDetails) { d.MoisturePercent = 8 }, ""},
		{"moisture at maximum", func(d *lot.FarmDetails) { d.MoisturePercent = 13 }, ""},
		{"moisture above maximum", func(d *lot.FarmDetails) { d.MoisturePercent = 13.01 }, exception.CodeStageInvalidMoisture},
		{"zero parchment weight", func(d *lot.FarmDetails) { d.ParchmentWeightKg = 0 }, exception.CodeStageInvalidParchmentWeight},
		{"negative parchment weight", func(d *lot.FarmDetails) { d.ParchmentWeightKg = -1 }, exception.CodeStageInvalidParchmentWeight},
		{"small parchment weight", func(d *lot.FarmDetails) { d.ParchmentWeightKg = 0.01 }, ""},
		{"zero value", func(d *lot.FarmDetails) { *d = lot.FarmDetails{} }, exception.CodeStageInvalidDryingMethod},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			details := validFarmDetails()
			tt.mutate(&details)

			err := details.Validate(stageDetailsEventDate())
			assertDetailsError(t, err, tt.wantCode)
		})
	}
}

func TestRoastingDetailsValidate(t *testing.T) {
	tests := []struct {
		name     string
		mutate   func(*lot.RoastingDetails)
		wantCode exception.Code
	}{
		{"valid", func(*lot.RoastingDetails) {}, ""},
		{"unknown roast level", func(d *lot.RoastingDetails) { d.RoastLevel = lot.RoastLevelUnknown }, exception.CodeStageInvalidRoastLevel},
		{"empty roast level", func(d *lot.RoastingDetails) { d.RoastLevel = "" }, exception.CodeStageInvalidRoastLevel},
		{"invalid roast level", func(d *lot.RoastingDetails) { d.RoastLevel = "burnt" }, exception.CodeStageInvalidRoastLevel},
		{"light roast", func(d *lot.RoastingDetails) { d.RoastLevel = lot.RoastLevelLight }, ""},
		{"dark roast", func(d *lot.RoastingDetails) { d.RoastLevel = lot.RoastLevelDark }, ""},
		{"duration below minimum", func(d *lot.RoastingDetails) { d.DurationMinutes = 0 }, exception.CodeStageInvalidRoastDuration},
		{"negative duration", func(d *lot.RoastingDetails) { d.DurationMinutes = -5 }, exception.CodeStageInvalidRoastDuration},
		{"duration at minimum", func(d *lot.RoastingDetails) { d.DurationMinutes = 1 }, ""},
		{"duration at maximum", func(d *lot.RoastingDetails) { d.DurationMinutes = 60 }, ""},
		{"duration above maximum", func(d *lot.RoastingDetails) { d.DurationMinutes = 61 }, exception.CodeStageInvalidRoastDuration},
		{"zero green weight", func(d *lot.RoastingDetails) { d.GreenWeightKg = 0 }, exception.CodeStageInvalidGreenWeight},
		{"negative green weight", func(d *lot.RoastingDetails) { d.GreenWeightKg = -10 }, exception.CodeStageInvalidGreenWeight},
		{"zero roasted weight", func(d *lot.RoastingDetails) { d.RoastedWeightKg = 0 }, exception.CodeStageInvalidRoastedWeight},
		{"negative roasted weight", func(d *lot.RoastingDetails) { d.RoastedWeightKg = -1 }, exception.CodeStageInvalidRoastedWeight},
		{"roasted weight equal to green", func(d *lot.RoastingDetails) { d.RoastedWeightKg = d.GreenWeightKg }, exception.CodeStageInvalidRoastedWeight},
		{"roasted weight above green", func(d *lot.RoastingDetails) { d.RoastedWeightKg = d.GreenWeightKg + 1 }, exception.CodeStageInvalidRoastedWeight},
		{"roasted weight just below green", func(d *lot.RoastingDetails) { d.RoastedWeightKg = d.GreenWeightKg - 0.01 }, ""},
		{"zero value", func(d *lot.RoastingDetails) { *d = lot.RoastingDetails{} }, exception.CodeStageInvalidRoastLevel},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			details := validRoastingDetails()
			tt.mutate(&details)

			err := details.Validate(stageDetailsEventDate())
			assertDetailsError(t, err, tt.wantCode)
		})
	}
}

func TestPackagingDetailsValidate(t *testing.T) {
	eventDate := stageDetailsEventDate()

	tests := []struct {
		name     string
		mutate   func(*lot.PackagingDetails)
		wantCode exception.Code
	}{
		{"valid", func(*lot.PackagingDetails) {}, ""},
		{"ground", func(d *lot.PackagingDetails) { d.Grind = lot.GrindGround }, ""},
		{"unknown grind", func(d *lot.PackagingDetails) { d.Grind = lot.GrindUnknown }, exception.CodeStageInvalidGrind},
		{"empty grind", func(d *lot.PackagingDetails) { d.Grind = "" }, exception.CodeStageInvalidGrind},
		{"invalid grind", func(d *lot.PackagingDetails) { d.Grind = "powder" }, exception.CodeStageInvalidGrind},
		{"package weight 250", func(d *lot.PackagingDetails) { d.PackageWeightG = 250 }, ""},
		{"package weight 340", func(d *lot.PackagingDetails) { d.PackageWeightG = 340 }, ""},
		{"package weight 500", func(d *lot.PackagingDetails) { d.PackageWeightG = 500 }, ""},
		{"package weight 1000", func(d *lot.PackagingDetails) { d.PackageWeightG = 1000 }, ""},
		{"package weight not allowed", func(d *lot.PackagingDetails) { d.PackageWeightG = 300 }, exception.CodeStageInvalidPackageWeight},
		{"zero package weight", func(d *lot.PackagingDetails) { d.PackageWeightG = 0 }, exception.CodeStageInvalidPackageWeight},
		{"negative package weight", func(d *lot.PackagingDetails) { d.PackageWeightG = -250 }, exception.CodeStageInvalidPackageWeight},
		{"zero units", func(d *lot.PackagingDetails) { d.Units = 0 }, exception.CodeStageInvalidUnit},
		{"negative units", func(d *lot.PackagingDetails) { d.Units = -1 }, exception.CodeStageInvalidUnit},
		{"one unit", func(d *lot.PackagingDetails) { d.Units = 1 }, ""},
		{"empty best before", func(d *lot.PackagingDetails) { d.BestBefore = "" }, exception.CodeStageInvalidBestBefore},
		{"blank best before", func(d *lot.PackagingDetails) { d.BestBefore = "   " }, exception.CodeStageInvalidBestBefore},
		{"best before with wrong format", func(d *lot.PackagingDetails) { d.BestBefore = "31/12/2030" }, exception.CodeStageInvalidBestBefore},
		{"best before with time", func(d *lot.PackagingDetails) { d.BestBefore = "2030-12-31T00:00:00Z" }, exception.CodeStageInvalidBestBefore},
		{"best before not a date", func(d *lot.PackagingDetails) { d.BestBefore = "soon" }, exception.CodeStageInvalidBestBefore},
		{"best before on the event date", func(d *lot.PackagingDetails) { d.BestBefore = eventDate.Format(time.DateOnly) }, exception.CodeStageInvalidBestBefore},
		{"best before before the event date", func(d *lot.PackagingDetails) { d.BestBefore = eventDate.AddDate(0, 0, -1).Format(time.DateOnly) }, exception.CodeStageInvalidBestBefore},
		{"best before the day after the event date", func(d *lot.PackagingDetails) { d.BestBefore = eventDate.AddDate(0, 0, 1).Format(time.DateOnly) }, ""},
		{"best before surrounded by spaces", func(d *lot.PackagingDetails) {
			d.BestBefore = "  " + eventDate.AddDate(0, 0, 10).Format(time.DateOnly) + "  "
		}, ""},
		{"zero value", func(d *lot.PackagingDetails) { *d = lot.PackagingDetails{} }, exception.CodeStageInvalidGrind},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			details := validPackagingDetails()
			tt.mutate(&details)

			err := details.Validate(eventDate)
			assertDetailsError(t, err, tt.wantCode)
		})
	}
}

func TestPackagingDetailsBestBeforeIgnoresEventTime(t *testing.T) {
	eventDate := time.Date(2026, 3, 10, 18, 45, 0, 0, time.UTC)
	details := validPackagingDetails()

	details.BestBefore = "2026-03-10"
	if err := details.Validate(eventDate); !exception.HasCode(err, exception.CodeStageInvalidBestBefore) {
		t.Errorf("same day: got %v, want code %q", err, exception.CodeStageInvalidBestBefore)
	}

	details.BestBefore = "2026-03-11"
	if err := details.Validate(eventDate); err != nil {
		t.Errorf("next day: unexpected error: %v", err)
	}
}

func TestStageDetailsRangeErrorsCarryLimits(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		wantMin any
		wantMax any
	}{
		{"drying days", func() error {
			d := validFarmDetails()
			d.DryingDays = 0
			return d.Validate(stageDetailsEventDate())
		}(), 1, 60},
		{"moisture", func() error {
			d := validFarmDetails()
			d.MoisturePercent = 0
			return d.Validate(stageDetailsEventDate())
		}(), 8.0, 13.0},
		{"roast duration", func() error {
			d := validRoastingDetails()
			d.DurationMinutes = 0
			return d.Validate(stageDetailsEventDate())
		}(), 1, 60},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ex, ok := exception.From(tt.err)
			if !ok {
				t.Fatalf("expected an exception, got %v", tt.err)
			}
			if ex.Params()["min"] != tt.wantMin || ex.Params()["max"] != tt.wantMax {
				t.Errorf("params = %v, want min=%v max=%v", ex.Params(), tt.wantMin, tt.wantMax)
			}
		})
	}
}

func assertDetailsError(t *testing.T, err error, wantCode exception.Code) {
	t.Helper()

	if wantCode == "" {
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		return
	}
	if !exception.HasCode(err, wantCode) {
		t.Errorf("got %v, want code %q", err, wantCode)
	}
}
