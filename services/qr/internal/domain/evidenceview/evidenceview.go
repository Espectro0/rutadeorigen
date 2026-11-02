package evidenceview

import (
	"time"

	"github.com/google/uuid"

	"rutadeorigen/qr/internal/domain/lotview"
	"rutadeorigen/qr/internal/platform/exception"
	"rutadeorigen/qr/internal/platform/helper"
)

var allowedContentTypes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/webp": true,
}

type EvidenceView struct {
	evidenceID   uuid.UUID
	lotID        uuid.UUID
	stage        lotview.Stage
	finalKey     string
	thumbnailKey string
	contentType  string
	processedAt  time.Time
}

var defaultEvidenceView = EvidenceView{
	evidenceID:   helper.DefaultUUID(),
	lotID:        helper.DefaultUUID(),
	stage:        lotview.StageUnknown,
	finalKey:     helper.EmptyString,
	thumbnailKey: helper.EmptyString,
	contentType:  helper.EmptyString,
	processedAt:  helper.DefaultDate(),
}

func Default() *EvidenceView {
	v := defaultEvidenceView
	return &v
}

type NewParams struct {
	EvidenceID   uuid.UUID
	LotID        uuid.UUID
	Stage        lotview.Stage
	FinalKey     string
	ThumbnailKey string
	ContentType  string
	ProcessedAt  time.Time
}

func New(p NewParams) (*EvidenceView, error) {
	finalKey := helper.ApplyTrim(p.FinalKey)
	thumbnailKey := helper.ApplyTrim(p.ThumbnailKey)
	contentType := helper.ApplyLower(helper.ApplyTrim(p.ContentType))

	invalid := helper.IsDefaultUUID(p.EvidenceID) ||
		helper.IsDefaultUUID(p.LotID) ||
		!p.Stage.IsValid() || p.Stage == lotview.StageNone ||
		helper.IsEmpty(finalKey) || helper.IsEmpty(thumbnailKey) ||
		!allowedContentTypes[contentType]
	if invalid {
		return Default(), exception.New(exception.CodeEventInvalidPayload)
	}

	processedAt := time.Now().UTC()
	if !helper.IsDefaultDate(p.ProcessedAt) {
		processedAt = p.ProcessedAt.UTC()
	}

	return &EvidenceView{
		evidenceID:   p.EvidenceID,
		lotID:        p.LotID,
		stage:        p.Stage,
		finalKey:     finalKey,
		thumbnailKey: thumbnailKey,
		contentType:  contentType,
		processedAt:  processedAt,
	}, nil
}

type RestoreParams = NewParams

func Restore(p RestoreParams) *EvidenceView {
	return &EvidenceView{
		evidenceID:   p.EvidenceID,
		lotID:        p.LotID,
		stage:        p.Stage,
		finalKey:     p.FinalKey,
		thumbnailKey: p.ThumbnailKey,
		contentType:  p.ContentType,
		processedAt:  p.ProcessedAt.UTC(),
	}
}

func (v *EvidenceView) EvidenceID() uuid.UUID  { return v.evidenceID }
func (v *EvidenceView) LotID() uuid.UUID       { return v.lotID }
func (v *EvidenceView) Stage() lotview.Stage   { return v.stage }
func (v *EvidenceView) FinalKey() string       { return v.finalKey }
func (v *EvidenceView) ThumbnailKey() string   { return v.thumbnailKey }
func (v *EvidenceView) ContentType() string    { return v.contentType }
func (v *EvidenceView) ProcessedAt() time.Time { return v.processedAt }
func (v *EvidenceView) IsDefault() bool        { return v == nil || helper.IsDefaultUUID(v.evidenceID) }
