package picture

import "rutadeorigen/processor/internal/platform/exception"

const (
	bytesPerMegabyte   = 1 << 20
	pixelsPerMegapixel = 1_000_000
)

type Limits struct {
	maxFileBytes  int64
	maxPixels     int64
	thumbnailSide int
}

func DefaultLimits() Limits { return Limits{} }

func NewLimits(maxFileBytes int64, maxMegapixels, thumbnailSide int) (Limits, error) {
	if maxFileBytes <= 0 || maxMegapixels <= 0 || thumbnailSide <= 0 {
		return DefaultLimits(), exception.New(exception.CodeInternal)
	}

	return Limits{
		maxFileBytes:  maxFileBytes,
		maxPixels:     int64(maxMegapixels) * pixelsPerMegapixel,
		thumbnailSide: thumbnailSide,
	}, nil
}

func (l Limits) IsDefault() bool { return l.maxFileBytes <= 0 }

func (l Limits) MaxFileBytes() int64 { return l.maxFileBytes }
func (l Limits) MaxMegapixels() int  { return int(l.maxPixels / pixelsPerMegapixel) }
func (l Limits) ThumbnailSide() int  { return l.thumbnailSide }

func (l Limits) CheckFileSize(sizeBytes int64) error {
	if sizeBytes > l.maxFileBytes {
		return exception.New(exception.CodeFileTooLarge).
			WithParams(map[string]any{"max": l.maxFileBytes / bytesPerMegabyte})
	}
	return nil
}

func (l Limits) CheckDimensions(d Dimensions) error {
	if d.IsDefault() {
		return exception.New(exception.CodeImageInvalidDimensions)
	}
	if d.Pixels() > l.maxPixels {
		return exception.New(exception.CodeImageTooManyPixels).
			WithParams(map[string]any{"max": l.MaxMegapixels()})
	}
	return nil
}
