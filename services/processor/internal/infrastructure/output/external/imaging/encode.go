package imaging

import (
	"bytes"
	"image"
	"image/jpeg"
	"image/png"

	"github.com/HugoSmits86/nativewebp"
	xdraw "golang.org/x/image/draw"

	"rutadeorigen/processor/internal/domain/picture"
	"rutadeorigen/processor/internal/platform/exception"
)

const jpegQuality = 90

func encodeInFormat(img image.Image, format picture.Format) ([]byte, error) {
	var buffer bytes.Buffer
	var err error

	switch format {
	case picture.FormatJPEG:
		err = jpeg.Encode(&buffer, img, &jpeg.Options{Quality: jpegQuality})
	case picture.FormatPNG:
		err = png.Encode(&buffer, img)
	case picture.FormatWebP:
		err = nativewebp.Encode(&buffer, img, &nativewebp.Options{})
	default:
		return nil, exception.New(exception.CodeImageUnsupportedFormat)
	}

	if err != nil {
		return nil, processingFailed("encode "+format.String(), err)
	}
	return buffer.Bytes(), nil
}

func encodeThumbnail(img image.Image, side int) ([]byte, error) {
	scaled, err := scaleToFit(img, side)
	if err != nil {
		return nil, err
	}
	return encodeInFormat(scaled, picture.FormatWebP)
}

func scaleToFit(img image.Image, side int) (image.Image, error) {
	bounds := img.Bounds()
	source, err := picture.NewDimensions(bounds.Dx(), bounds.Dy())
	if err != nil {
		return nil, err
	}

	target := source.FitWithin(side)
	scaled := image.NewNRGBA(image.Rect(0, 0, target.Width(), target.Height()))
	xdraw.CatmullRom.Scale(scaled, scaled.Bounds(), img, bounds, xdraw.Over, nil)
	return scaled, nil
}

func processingFailed(technical string, cause error) error {
	return exception.New(exception.CodeImageProcessingFailed).
		WithTechnical(technical).
		WithCause(cause)
}
