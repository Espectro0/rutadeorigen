package lot

import (
	"time"

	"github.com/google/uuid"

	"rutadeorigen/traceability/internal/platform/helper"
)

const (
	codeMinLength  = 3
	codeMaxLength  = 30
	initialVersion = 1
)

type Lot struct {
	id           uuid.UUID
	farmID       uuid.UUID
	custodianID  uuid.UUID
	code         string
	variety      string
	process      Process
	harvestDate  time.Time
	currentStage Stage
	status       Status
	voidReason   string
	version      int
	createdAt    time.Time
}

var defaultLot = Lot{
	id:           helper.DefaultUUID(),
	farmID:       helper.DefaultUUID(),
	custodianID:  helper.DefaultUUID(),
	code:         helper.EmptyString,
	variety:      helper.EmptyString,
	process:      ProcessUnknown,
	harvestDate:  helper.DefaultDate(),
	currentStage: StageNone,
	status:       StatusUnknown,
	voidReason:   helper.EmptyString,
	version:      helper.DefaultInt(),
	createdAt:    helper.DefaultDate(),
}

func Default() *Lot {
	l := defaultLot
	return &l
}

func New(farmID, custodianID uuid.UUID, code, variety string, process Process, harvestDate time.Time) (*Lot, error) {
	code = helper.ApplyTrim(code)
	variety = helper.ApplyTrim(variety)

	if helper.IsDefaultUUID(farmID) {
		return Default(), ErrInvalidFarm
	}
	if helper.IsDefaultUUID(custodianID) {
		return Default(), ErrInvalidCustodian
	}
	if !helper.LengthIsValid(code, codeMinLength, codeMaxLength) {
		return Default(), ErrInvalidCode
	}
	if helper.IsEmpty(variety) {
		return Default(), ErrInvalidVariety
	}
	if !process.IsValid() {
		return Default(), ErrInvalidProcess
	}
	if helper.IsDefaultDate(harvestDate) || harvestDate.After(time.Now()) {
		return Default(), ErrInvalidHarvestDate
	}

	return &Lot{
		id:           uuid.New(),
		farmID:       farmID,
		custodianID:  custodianID,
		code:         code,
		variety:      variety,
		process:      process,
		harvestDate:  harvestDate,
		currentStage: StageNone,
		status:       StatusActive,
		voidReason:   helper.EmptyString,
		version:      initialVersion,
		createdAt:    time.Now().UTC(),
	}, nil
}

type RestoreParams struct {
	ID           uuid.UUID
	FarmID       uuid.UUID
	CustodianID  uuid.UUID
	Code         string
	Variety      string
	Process      Process
	HarvestDate  time.Time
	CurrentStage Stage
	Status       Status
	VoidReason   string
	Version      int
	CreatedAt    time.Time
}

func Restore(p RestoreParams) *Lot {
	return &Lot{
		id:           p.ID,
		farmID:       p.FarmID,
		custodianID:  p.CustodianID,
		code:         p.Code,
		variety:      p.Variety,
		process:      p.Process,
		harvestDate:  p.HarvestDate,
		currentStage: p.CurrentStage,
		status:       p.Status,
		voidReason:   p.VoidReason,
		version:      p.Version,
		createdAt:    p.CreatedAt,
	}
}

func (l *Lot) IsActive() bool  { return l.status == StatusActive }
func (l *Lot) HasEvents() bool { return l.currentStage != StageNone }

func (l *Lot) CanBeDeleted() error {
	if l.HasEvents() {
		return ErrHasEvents
	}
	return nil
}

func (l *Lot) Void(reason string) error {
	if !l.IsActive() {
		return ErrVoided
	}
	if helper.IsEmpty(reason) {
		return ErrEmptyVoidReason
	}

	l.status = StatusVoided
	l.voidReason = helper.ApplyTrim(reason)
	l.version++
	return nil
}

func (l *Lot) ID() uuid.UUID          { return l.id }
func (l *Lot) FarmID() uuid.UUID      { return l.farmID }
func (l *Lot) CustodianID() uuid.UUID { return l.custodianID }
func (l *Lot) Code() string           { return l.code }
func (l *Lot) Variety() string        { return l.variety }
func (l *Lot) Process() Process       { return l.process }
func (l *Lot) HarvestDate() time.Time { return l.harvestDate }
func (l *Lot) CurrentStage() Stage    { return l.currentStage }
func (l *Lot) Status() Status         { return l.status }
func (l *Lot) VoidReason() string     { return l.voidReason }
func (l *Lot) Version() int           { return l.version }
func (l *Lot) CreatedAt() time.Time   { return l.createdAt }

func (l *Lot) IsDefault() bool {
	return l == nil || helper.IsDefaultUUID(l.id)
}
