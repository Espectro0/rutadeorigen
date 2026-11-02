package imaging

import (
	"bytes"
	"context"
	"errors"
	"image"
	_ "image/jpeg"
	_ "image/png"

	_ "golang.org/x/image/webp"

	"rutadeorigen/processor/internal/application/output/ports"
	"rutadeorigen/processor/internal/domain/picture"
	"rutadeorigen/processor/internal/platform/exception"
)

var _ ports.ImageProcessor = (*Processor)(nil)

type Processor struct{}

func New() *Processor {
	return &Processor{}
}

func (p *Processor) Inspect(_ context.Context, data []byte) (picture.Info, error) {
	config, formatName, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return picture.DefaultInfo(), classifyDecodeError(err)
	}
	return picture.NewInfo(formatName, config.Width, config.Height)
}

func (p *Processor) Process(ctx context.Context, data []byte, info picture.Info, thumbnailSide int) (ports.ProcessedImage, error) {
	failed := ports.ProcessedImage{}

	if err := ctx.Err(); err != nil {
		return failed, exception.Wrap(err, "image processing canceled")
	}

	decoded, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return failed, classifyDecodeError(err)
	}

	upright := applyOrientation(decoded, readOrientation(info.Format(), data))

	final, err := encodeInFormat(upright, info.Format())
	if err != nil {
		return failed, err
	}

	thumbnail, err := encodeThumbnail(upright, thumbnailSide)
	if err != nil {
		return failed, err
	}

	return ports.ProcessedImage{Final: final, Thumbnail: thumbnail}, nil
}

func classifyDecodeError(err error) error {
	if errors.Is(err, image.ErrFormat) {
		return exception.New(exception.CodeImageUnsupportedFormat).WithCause(err)
	}
	return exception.New(exception.CodeImageCorrupt).WithCause(err)
}
