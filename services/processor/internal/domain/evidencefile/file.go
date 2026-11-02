package evidencefile

import (
	"github.com/google/uuid"

	"rutadeorigen/processor/internal/domain/picture"
	"rutadeorigen/processor/internal/platform/exception"
	"rutadeorigen/processor/internal/platform/helper"
)

const (
	finalImageName = "imagen"
	thumbnailName  = "miniatura.webp"
)

type File struct {
	evidenceID    uuid.UUID
	lotID         uuid.UUID
	quarantineKey string
	sizeBytes     int64
}

var defaultFile = File{
	evidenceID:    helper.DefaultUUID(),
	lotID:         helper.DefaultUUID(),
	quarantineKey: helper.EmptyString,
	sizeBytes:     0,
}

func Default() *File {
	f := defaultFile
	return &f
}

type NewParams struct {
	EvidenceID    uuid.UUID
	LotID         uuid.UUID
	QuarantineKey string
	SizeBytes     int64
}

func New(p NewParams) (*File, error) {
	if helper.IsDefaultUUID(p.EvidenceID) || helper.IsDefaultUUID(p.LotID) || p.SizeBytes <= 0 {
		return Default(), exception.New(exception.CodeEventInvalidPayload)
	}

	f := &File{
		evidenceID:    p.EvidenceID,
		lotID:         p.LotID,
		quarantineKey: helper.ApplyTrim(p.QuarantineKey),
		sizeBytes:     p.SizeBytes,
	}

	if f.quarantineKey != f.ExpectedQuarantineKey() {
		return Default(), exception.New(exception.CodeEvidenceKeyMismatch)
	}
	return f, nil
}

func (f *File) ExpectedQuarantineKey() string {
	return f.lotID.String() + "/" + f.evidenceID.String()
}

func (f *File) MaxDownloadBytes() int64 { return f.sizeBytes + 1 }

func (f *File) CheckUploadedSize(actual int64) error {
	if actual != f.sizeBytes {
		return exception.New(exception.CodeFileSizeMismatch)
	}
	return nil
}

func (f *File) FinalKey(format picture.Format) string {
	return f.lotID.String() + "/" + f.evidenceID.String() + "/" + finalImageName + "." + format.Extension()
}

func (f *File) ThumbnailKey() string {
	return f.lotID.String() + "/" + f.evidenceID.String() + "/" + thumbnailName
}

func (f *File) EvidenceID() uuid.UUID { return f.evidenceID }
func (f *File) LotID() uuid.UUID      { return f.lotID }
func (f *File) QuarantineKey() string { return f.quarantineKey }
func (f *File) SizeBytes() int64      { return f.sizeBytes }

func (f *File) IsDefault() bool {
	return f == nil || helper.IsDefaultUUID(f.evidenceID)
}
