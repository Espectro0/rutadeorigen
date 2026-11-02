package imaging

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/jpeg"
	"image/png"
	"io"

	"golang.org/x/image/webp"

	"rutadeorigen/processor/internal/application/output/ports"
	"rutadeorigen/processor/internal/domain/picture"
	"rutadeorigen/processor/internal/platform/exception"
)

var _ ports.ImageProcessor = (*Processor)(nil)

var decoders = map[picture.Format]func(io.Reader) (image.Image, error){
	picture.FormatJPEG: jpeg.Decode,
	picture.FormatPNG:  png.Decode,
	picture.FormatWebP: webp.Decode,
}

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

	decode, supported := decoders[info.Format()]
	if !supported {
		return failed, exception.New(exception.CodeImageUnsupportedFormat)
	}

	decoded, err := decode(bytes.NewReader(data))
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
