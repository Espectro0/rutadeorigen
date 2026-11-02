package evidence

import (
	"path"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"

	"rutadeorigen/traceability/internal/domain/lot"
	"rutadeorigen/traceability/internal/platform/exception"
	"rutadeorigen/traceability/internal/platform/helper"
)

const (
	fileNameMinLength = 1
	fileNameMaxLength = 255
)

type Evidence struct {
	id              uuid.UUID
	lotID           uuid.UUID
	stage           lot.Stage
	uploadedBy      uuid.UUID
	fileName        string
	contentType     ContentType
	sizeBytes       int64
	status          Status
	sha256          string
	finalKey        string
	thumbnailKey    string
	rejectionReason string
	createdAt       time.Time
	processedAt     time.Time
}

var defaultEvidence = Evidence{
	id:              helper.DefaultUUID(),
	lotID:           helper.DefaultUUID(),
	stage:           lot.StageUnknown,
	uploadedBy:      helper.DefaultUUID(),
	fileName:        helper.EmptyString,
	contentType:     ContentTypeUnknown,
	sizeBytes:       0,
	status:          StatusUnknown,
	sha256:          helper.EmptyString,
	finalKey:        helper.EmptyString,
	thumbnailKey:    helper.EmptyString,
	rejectionReason: helper.EmptyString,
	createdAt:       helper.DefaultDate(),
	processedAt:     helper.DefaultDate(),
}

func Default() *Evidence {
	e := defaultEvidence
	return &e
}

type NewParams struct {
	LotID       uuid.UUID
	Stage       lot.Stage
	UploadedBy  uuid.UUID
	FileName    string
	ContentType string
	SizeBytes   int64
	MaxBytes    int64
}

func New(p NewParams) (*Evidence, error) {
	if helper.IsDefaultUUID(p.LotID) || helper.IsDefaultUUID(p.UploadedBy) {
		return Default(), exception.New(exception.CodeInternal).
			WithTechnical("evidence requires a lot and an uploader")
	}
	if p.Stage == lot.StageNone {
		return Default(), exception.New(exception.CodeEvidenceLotWithoutStage)
	}
	if !p.Stage.IsValid() {
		return Default(), exception.New(exception.CodeLotInvalidStage)
	}

	name := cleanFileName(p.FileName)
	if !helper.LengthIsValid(name, fileNameMinLength, fileNameMaxLength) {
		return Default(), exception.New(exception.CodeEvidenceInvalidFileName).
			WithParams(map[string]any{"min": fileNameMinLength, "max": fileNameMaxLength})
	}

	contentType := ParseContentType(p.ContentType)
	if !contentType.IsValid() {
		return Default(), exception.New(exception.CodeEvidenceInvalidContentType)
	}

	if p.SizeBytes <= 0 {
		return Default(), exception.New(exception.CodeEvidenceEmptyFile)
	}
	if p.SizeBytes > p.MaxBytes {
		return Default(), exception.New(exception.CodeEvidenceTooLarge).
			WithParams(map[string]any{"max": p.MaxBytes / (1 << 20)})
	}

	return &Evidence{
		id:              uuid.New(),
		lotID:           p.LotID,
		stage:           p.Stage,
		uploadedBy:      p.UploadedBy,
		fileName:        name,
		contentType:     contentType,
		sizeBytes:       p.SizeBytes,
		status:          StatusPendingUpload,
		sha256:          helper.EmptyString,
		finalKey:        helper.EmptyString,
		thumbnailKey:    helper.EmptyString,
		rejectionReason: helper.EmptyString,
		createdAt:       time.Now().UTC().Truncate(time.Microsecond),
		processedAt:     helper.DefaultDate(),
	}, nil
}

type RestoreParams struct {
	ID              uuid.UUID
	LotID           uuid.UUID
	Stage           lot.Stage
	UploadedBy      uuid.UUID
	FileName        string
	ContentType     ContentType
	SizeBytes       int64
	Status          Status
	SHA256          string
	FinalKey        string
	ThumbnailKey    string
	RejectionReason string
	CreatedAt       time.Time
	ProcessedAt     time.Time
}

func Restore(p RestoreParams) *Evidence {
	return &Evidence{
		id:              p.ID,
		lotID:           p.LotID,
		stage:           p.Stage,
		uploadedBy:      p.UploadedBy,
		fileName:        p.FileName,
		contentType:     p.ContentType,
		sizeBytes:       p.SizeBytes,
		status:          p.Status,
		sha256:          p.SHA256,
		finalKey:        p.FinalKey,
		thumbnailKey:    p.ThumbnailKey,
		rejectionReason: p.RejectionReason,
		createdAt:       p.CreatedAt,
		processedAt:     p.ProcessedAt,
	}
}

// MarkUploaded confirms that the file reached quarantine. It is idempotent
// while the evidence has not been resolved by the processor.
func (e *Evidence) MarkUploaded() error {
	if e.status.IsFinal() {
		return exception.New(exception.CodeEvidenceAlreadyFinal)
	}
	e.status = StatusUploaded
	return nil
}

func (e *Evidence) MarkProcessed(sha256, finalKey, thumbnailKey string, at time.Time) error {
	if e.status.IsFinal() {
		return exception.New(exception.CodeEvidenceAlreadyFinal)
	}
	if e.status != StatusUploaded {
		return exception.New(exception.CodeEvidenceInvalidResult)
	}

	sha256 = helper.ApplyTrim(sha256)
	finalKey = helper.ApplyTrim(finalKey)
	thumbnailKey = helper.ApplyTrim(thumbnailKey)
	if helper.IsEmpty(sha256) || helper.IsEmpty(finalKey) || helper.IsEmpty(thumbnailKey) || helper.IsDefaultDate(at) {
		return exception.New(exception.CodeEvidenceInvalidResult)
	}

	e.status = StatusProcessed
	e.sha256 = sha256
	e.finalKey = finalKey
	e.thumbnailKey = thumbnailKey
	e.processedAt = at.UTC()
	return nil
}

func (e *Evidence) MarkRejected(reason string, at time.Time) error {
	if e.status.IsFinal() {
		return exception.New(exception.CodeEvidenceAlreadyFinal)
	}
	if e.status != StatusUploaded {
		return exception.New(exception.CodeEvidenceInvalidResult)
	}

	reason = helper.ApplyTrim(reason)
	if helper.IsEmpty(reason) || helper.IsDefaultDate(at) {
		return exception.New(exception.CodeEvidenceInvalidResult)
	}

	e.status = StatusRejected
	e.rejectionReason = reason
	e.processedAt = at.UTC()
	return nil
}

// QuarantineKey never uses the user's file name: only identifiers.
func (e *Evidence) QuarantineKey() string {
	return e.lotID.String() + "/" + e.id.String()
}

func (e *Evidence) IsUploadedBy(actorID uuid.UUID) bool {
	return !helper.IsDefaultUUID(actorID) && e.uploadedBy == actorID
}

func (e *Evidence) ID() uuid.UUID            { return e.id }
func (e *Evidence) LotID() uuid.UUID         { return e.lotID }
func (e *Evidence) Stage() lot.Stage         { return e.stage }
func (e *Evidence) UploadedBy() uuid.UUID    { return e.uploadedBy }
func (e *Evidence) FileName() string         { return e.fileName }
func (e *Evidence) ContentType() ContentType { return e.contentType }
func (e *Evidence) SizeBytes() int64         { return e.sizeBytes }
func (e *Evidence) Status() Status           { return e.status }
func (e *Evidence) SHA256() string           { return e.sha256 }
func (e *Evidence) FinalKey() string         { return e.finalKey }
func (e *Evidence) ThumbnailKey() string     { return e.thumbnailKey }
func (e *Evidence) RejectionReason() string  { return e.rejectionReason }
func (e *Evidence) CreatedAt() time.Time     { return e.createdAt }
func (e *Evidence) ProcessedAt() time.Time   { return e.processedAt }

func (e *Evidence) IsDefault() bool {
	return e == nil || helper.IsDefaultUUID(e.id)
}

func cleanFileName(value string) string {
	value = strings.ReplaceAll(value, "\\", "/")
	value = path.Base(helper.ApplyTrim(value))
	if value == "." || value == "/" {
		return helper.EmptyString
	}

	cleaned := strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, value)
	cleaned = helper.ApplyTrim(cleaned)

	if utf8.RuneCountInString(cleaned) > fileNameMaxLength {
		runes := []rune(cleaned)
		cleaned = string(runes[:fileNameMaxLength])
	}
	return cleaned
}
