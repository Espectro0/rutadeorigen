package lotview

import (
	"time"

	"github.com/google/uuid"

	"rutadeorigen/qr/internal/platform/exception"
	"rutadeorigen/qr/internal/platform/helper"
)

const (
	codeMinLength = 3
	codeMaxLength = 30
	minAltitude   = 0
	maxAltitude   = 3000
)

type LotView struct {
	lotID        uuid.UUID
	code         string
	variety      string
	process      Process
	harvestDate  time.Time
	farmID       uuid.UUID
	farmName     string
	farmAltitude int
	producerID   uuid.UUID
	producerName string
	custodianID  uuid.UUID
	currentStage Stage
	status       Status
	voidReason   string
	version      int
	createdAt    time.Time
	updatedAt    time.Time
}

var defaultLotView = LotView{
	lotID:        helper.DefaultUUID(),
	code:         helper.EmptyString,
	variety:      helper.EmptyString,
	process:      ProcessUnknown,
	harvestDate:  helper.DefaultDate(),
	farmID:       helper.DefaultUUID(),
	farmName:     helper.EmptyString,
	farmAltitude: helper.DefaultInt(),
	producerID:   helper.DefaultUUID(),
	producerName: helper.EmptyString,
	custodianID:  helper.DefaultUUID(),
	currentStage: StageNone,
	status:       StatusUnknown,
	voidReason:   helper.EmptyString,
	version:      helper.DefaultInt(),
	createdAt:    helper.DefaultDate(),
	updatedAt:    helper.DefaultDate(),
}

func Default() *LotView {
	v := defaultLotView
	return &v
}

type NewParams struct {
	LotID        uuid.UUID
	Code         string
	Variety      string
	Process      Process
	HarvestDate  time.Time
	FarmID       uuid.UUID
	FarmName     string
	FarmAltitude int
	ProducerID   uuid.UUID
	ProducerName string
	CustodianID  uuid.UUID
	CurrentStage Stage
	Status       Status
	VoidReason   string
	Version      int
	CreatedAt    time.Time
}

func New(p NewParams) (*LotView, error) {
	code := helper.ApplyTrim(p.Code)
	variety := helper.ApplyTrim(p.Variety)
	farmName := helper.ApplyTrim(p.FarmName)
	producerName := helper.ApplyTrim(p.ProducerName)

	if helper.IsDefaultUUID(p.LotID) {
		return Default(), exception.New(exception.CodeLotViewInvalidID)
	}
	if !helper.LengthIsValid(code, codeMinLength, codeMaxLength) {
		return Default(), exception.New(exception.CodeLotViewInvalidCode).
			WithParams(map[string]any{"min": codeMinLength, "max": codeMaxLength})
	}
	if helper.IsEmpty(variety) {
		return Default(), exception.New(exception.CodeLotViewInvalidVariety)
	}
	if !p.Process.IsValid() {
		return Default(), exception.New(exception.CodeLotViewInvalidProcess)
	}
	if helper.IsDefaultDate(p.HarvestDate) {
		return Default(), exception.New(exception.CodeLotViewInvalidHarvest)
	}
	if helper.IsDefaultUUID(p.FarmID) || helper.IsEmpty(farmName) ||
		!helper.IsInRange(p.FarmAltitude, minAltitude, maxAltitude) {
		return Default(), exception.New(exception.CodeLotViewInvalidFarm)
	}
	if helper.IsDefaultUUID(p.ProducerID) || helper.IsEmpty(producerName) {
		return Default(), exception.New(exception.CodeLotViewInvalidProducer)
	}
	if helper.IsDefaultUUID(p.CustodianID) {
		return Default(), exception.New(exception.CodeLotViewInvalidCustodian)
	}
	if !p.CurrentStage.IsValid() {
		return Default(), exception.New(exception.CodeLotViewInvalidStage)
	}
	if !p.Status.IsValid() {
		return Default(), exception.New(exception.CodeLotViewInvalidStatus)
	}
	if p.Version <= 0 {
		return Default(), exception.New(exception.CodeLotViewInvalidVersion)
	}

	now := time.Now().UTC()
	createdAt := now
	if !helper.IsDefaultDate(p.CreatedAt) {
		createdAt = p.CreatedAt.UTC()
	}

	return &LotView{
		lotID:        p.LotID,
		code:         code,
		variety:      variety,
		process:      p.Process,
		harvestDate:  helper.DateUTC(p.HarvestDate),
		farmID:       p.FarmID,
		farmName:     farmName,
		farmAltitude: p.FarmAltitude,
		producerID:   p.ProducerID,
		producerName: producerName,
		custodianID:  p.CustodianID,
		currentStage: p.CurrentStage,
		status:       p.Status,
		voidReason:   helper.ApplyTrim(p.VoidReason),
		version:      p.Version,
		createdAt:    createdAt,
		updatedAt:    now,
	}, nil
}

type RestoreParams struct {
	LotID        uuid.UUID
	Code         string
	Variety      string
	Process      Process
	HarvestDate  time.Time
	FarmID       uuid.UUID
	FarmName     string
	FarmAltitude int
	ProducerID   uuid.UUID
	ProducerName string
	CustodianID  uuid.UUID
	CurrentStage Stage
	Status       Status
	VoidReason   string
	Version      int
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func Restore(p RestoreParams) *LotView {
	return &LotView{
		lotID:        p.LotID,
		code:         p.Code,
		variety:      p.Variety,
		process:      p.Process,
		harvestDate:  p.HarvestDate,
		farmID:       p.FarmID,
		farmName:     p.FarmName,
		farmAltitude: p.FarmAltitude,
		producerID:   p.ProducerID,
		producerName: p.ProducerName,
		custodianID:  p.CustodianID,
		currentStage: p.CurrentStage,
		status:       p.Status,
		voidReason:   p.VoidReason,
		version:      p.Version,
		createdAt:    p.CreatedAt,
		updatedAt:    p.UpdatedAt,
	}
}

func (v *LotView) IsActive() bool { return v.status == StatusActive }

func (v *LotView) IsPackaged() bool { return v.currentStage == StagePackaging }

func (v *LotView) IsNewerThan(other *LotView) bool {
	return other.IsDefault() || v.version > other.version
}

func (v *LotView) CanGenerateImage() error {
	if !v.IsActive() || !v.IsPackaged() {
		return exception.New(exception.CodeLotViewImageNotAvailable)
	}
	return nil
}

func (v *LotView) LotID() uuid.UUID       { return v.lotID }
func (v *LotView) Code() string           { return v.code }
func (v *LotView) Variety() string        { return v.variety }
func (v *LotView) Process() Process       { return v.process }
func (v *LotView) HarvestDate() time.Time { return v.harvestDate }
func (v *LotView) FarmID() uuid.UUID      { return v.farmID }
func (v *LotView) FarmName() string       { return v.farmName }
func (v *LotView) FarmAltitude() int      { return v.farmAltitude }
func (v *LotView) ProducerID() uuid.UUID  { return v.producerID }
func (v *LotView) ProducerName() string   { return v.producerName }
func (v *LotView) CustodianID() uuid.UUID { return v.custodianID }
func (v *LotView) CurrentStage() Stage    { return v.currentStage }
func (v *LotView) Status() Status         { return v.status }
func (v *LotView) VoidReason() string     { return v.voidReason }
func (v *LotView) Version() int           { return v.version }
func (v *LotView) CreatedAt() time.Time   { return v.createdAt }
func (v *LotView) UpdatedAt() time.Time   { return v.updatedAt }

func (v *LotView) IsDefault() bool {
	return v == nil || helper.IsDefaultUUID(v.lotID)
}
