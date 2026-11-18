package stageview

import (
	"strings"
	"time"

	"github.com/google/uuid"

	"rutadeorigen/qr/internal/domain/lotview"
	"rutadeorigen/qr/internal/platform/exception"
	"rutadeorigen/qr/internal/platform/helper"
)

type Details struct {
	DryingMethod         string
	DryingDays           int
	RoastLevel           string
	RoastDurationMinutes int
	Grind                string
	PackageWeightGrams   int
	BestBefore           string
}

type StageView struct {
	lotID     uuid.UUID
	stage     lotview.Stage
	eventDate time.Time
	details   Details
}

var defaultStageView = StageView{
	lotID:     helper.DefaultUUID(),
	stage:     lotview.StageUnknown,
	eventDate: helper.DefaultDate(),
	details:   Details{},
}

func Default() *StageView {
	v := defaultStageView
	return &v
}

type NewParams struct {
	LotID     uuid.UUID
	Stage     lotview.Stage
	EventDate time.Time
	Details   Details
}

func New(p NewParams) (*StageView, error) {
	if !p.Stage.IsValid() || p.Stage == lotview.StageNone {
		return Default(), exception.New(exception.CodeLotViewInvalidStage)
	}
	if helper.IsDefaultUUID(p.LotID) || helper.IsDefaultDate(p.EventDate) {
		return Default(), exception.New(exception.CodeEventInvalidPayload)
	}

	return &StageView{
		lotID:     p.LotID,
		stage:     p.Stage,
		eventDate: helper.DateUTC(p.EventDate),
		details:   normalize(p.Details),
	}, nil
}

type RestoreParams = NewParams

func Restore(p RestoreParams) *StageView {
	return &StageView{
		lotID:     p.LotID,
		stage:     p.Stage,
		eventDate: p.EventDate.UTC(),
		details:   p.Details,
	}
}

func normalize(d Details) Details {
	return Details{
		DryingMethod:         strings.ToLower(helper.ApplyTrim(d.DryingMethod)),
		DryingDays:           max(d.DryingDays, 0),
		RoastLevel:           strings.ToLower(helper.ApplyTrim(d.RoastLevel)),
		RoastDurationMinutes: max(d.RoastDurationMinutes, 0),
		Grind:                strings.ToLower(helper.ApplyTrim(d.Grind)),
		PackageWeightGrams:   max(d.PackageWeightGrams, 0),
		BestBefore:           helper.ApplyTrim(d.BestBefore),
	}
}

func (v *StageView) LotID() uuid.UUID     { return v.lotID }
func (v *StageView) Stage() lotview.Stage { return v.stage }
func (v *StageView) EventDate() time.Time { return v.eventDate }
func (v *StageView) Details() Details     { return v.details }
func (v *StageView) IsDefault() bool      { return v == nil || helper.IsDefaultUUID(v.lotID) }
