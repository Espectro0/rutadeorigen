package lot

import (
	"strings"
	"time"

	"rutadeorigen/traceability/internal/platform/exception"
	"rutadeorigen/traceability/internal/platform/helper"
)

type StageDetails interface {
	Stage() Stage
	Validate(eventDate time.Time) error
}

type DryingMethod string

const (
	DryingMethodUnknown    DryingMethod = "unknown"
	DryingMethodSunPatio   DryingMethod = "sun_patio"
	DryingMethodAfricanBed DryingMethod = "african_bed"
	DryingMethodGreenhouse DryingMethod = "greenhouse"
	DryingMethodMechanical DryingMethod = "mechanical"
)

func ParseDryingMethod(value string) DryingMethod {
	switch v := DryingMethod(strings.ToLower(helper.ApplyTrim(value))); v {
	case DryingMethodSunPatio, DryingMethodAfricanBed, DryingMethodGreenhouse, DryingMethodMechanical:
		return v
	default:
		return DryingMethodUnknown
	}
}

func (v DryingMethod) IsValid() bool  { return ParseDryingMethod(string(v)) != DryingMethodUnknown }
func (v DryingMethod) String() string { return string(v) }

type RoastLevel string

const (
	RoastLevelUnknown RoastLevel = "unknown"
	RoastLevelLight   RoastLevel = "light"
	RoastLevelMedium  RoastLevel = "medium"
	RoastLevelDark    RoastLevel = "dark"
)

func ParseRoastLevel(value string) RoastLevel {
	switch v := RoastLevel(strings.ToLower(helper.ApplyTrim(value))); v {
	case RoastLevelLight, RoastLevelMedium, RoastLevelDark:
		return v
	default:
		return RoastLevelUnknown
	}
}

func (v RoastLevel) IsValid() bool  { return ParseRoastLevel(string(v)) != RoastLevelUnknown }
func (v RoastLevel) String() string { return string(v) }

type Grind string

const (
	GrindUnknown   Grind = "unknown"
	GrindWholeBean Grind = "whole_bean"
	GrindGround    Grind = "ground"
)

func ParseGrind(value string) Grind {
	switch v := Grind(strings.ToLower(helper.ApplyTrim(value))); v {
	case GrindWholeBean, GrindGround:
		return v
	default:
		return GrindUnknown
	}
}

func (v Grind) IsValid() bool  { return ParseGrind(string(v)) != GrindUnknown }
func (v Grind) String() string { return string(v) }

const (
	dryingDaysMin, dryingDaysMax     = 1, 60
	moistureMin, moistureMax         = 8.0, 13.0
	roastMinutesMin, roastMinutesMax = 1, 60
)

var allowedPackageWeights = map[int]bool{250: true, 340: true, 500: true, 1000: true}

type FarmDetails struct {
	DryingMethod      DryingMethod `json:"drying_method"`
	DryingDays        int          `json:"drying_days"`
	MoisturePercent   float64      `json:"moisture_percent"`
	ParchmentWeightKg float64      `json:"parchment_weight_kg"`
}

func (FarmDetails) Stage() Stage { return StageFarm }

func (d FarmDetails) Validate(_ time.Time) error {
	if !d.DryingMethod.IsValid() {
		return exception.New(exception.CodeStageInvalidDryingMethod)
	}
	if d.DryingDays < dryingDaysMin || d.DryingDays > dryingDaysMax {
		return exception.New(exception.CodeStageInvalidDryingDays).
			WithParams(map[string]any{"min": dryingDaysMin, "max": dryingDaysMax})
	}
	if d.MoisturePercent < moistureMin || d.MoisturePercent > moistureMax {
		return exception.New(exception.CodeStageInvalidMoisture).
			WithParams(map[string]any{"min": moistureMin, "max": moistureMax})
	}
	if d.ParchmentWeightKg <= 0 {
		return exception.New(exception.CodeStageInvalidParchmentWeight)
	}
	return nil
}

type RoastingDetails struct {
	RoastLevel      RoastLevel `json:"roast_level"`
	DurationMinutes int        `json:"duration_minutes"`
	GreenWeightKg   float64    `json:"green_weight_kg"`
	RoastedWeightKg float64    `json:"roasted_weight_kg"`
}

func (RoastingDetails) Stage() Stage { return StageRoasting }

func (d RoastingDetails) Validate(_ time.Time) error {
	if !d.RoastLevel.IsValid() {
		return exception.New(exception.CodeStageInvalidRoastLevel)
	}
	if d.DurationMinutes < roastMinutesMin || d.DurationMinutes > roastMinutesMax {
		return exception.New(exception.CodeStageInvalidRoastDuration).
			WithParams(map[string]any{"min": roastMinutesMin, "max": roastMinutesMax})
	}
	if d.GreenWeightKg <= 0 {
		return exception.New(exception.CodeStageInvalidGreenWeight)
	}
	if d.RoastedWeightKg <= 0 || d.RoastedWeightKg >= d.GreenWeightKg {
		return exception.New(exception.CodeStageInvalidRoastedWeight)
	}
	return nil
}

type PackagingDetails struct {
	Grind          Grind  `json:"grind"`
	PackageWeightG int    `json:"package_weight_g"`
	Units          int    `json:"units"`
	BestBefore     string `json:"best_before"`
}

func (PackagingDetails) Stage() Stage { return StagePackaging }

func (d PackagingDetails) Validate(eventDate time.Time) error {
	if !d.Grind.IsValid() {
		return exception.New(exception.CodeStageInvalidGrind)
	}
	if !allowedPackageWeights[d.PackageWeightG] {
		return exception.New(exception.CodeStageInvalidPackageWeight)
	}
	if d.Units <= 0 {
		return exception.New(exception.CodeStageInvalidUnit)
	}

	bestBefore, err := time.Parse(time.DateOnly, helper.ApplyTrim(d.BestBefore))
	if err != nil || !bestBefore.After(helper.DateUTC(eventDate)) {
		return exception.New(exception.CodeStageInvalidBestBefore)
	}
	return nil
}
