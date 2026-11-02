package imaging

import (
	"bytes"
	"context"
	"encoding/binary"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"

	"github.com/HugoSmits86/nativewebp"

	"rutadeorigen/processor/internal/domain/picture"
	"rutadeorigen/processor/internal/platform/exception"
)

const thumbnailSide = 400

func gradient(width, height int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.SetNRGBA(x, y, color.NRGBA{R: uint8(x % 256), G: uint8(y % 256), B: 128, A: 255})
		}
	}
	return img
}

func encodedImage(t *testing.T, format picture.Format, width, height int) []byte {
	t.Helper()
	data, err := encodeInFormat(gradient(width, height), format)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func jpegWithOrientation(t *testing.T, width, height, orientation int) []byte {
	t.Helper()

	var plain bytes.Buffer
	if err := jpeg.Encode(&plain, gradient(width, height), nil); err != nil {
		t.Fatal(err)
	}
	encoded := plain.Bytes()

	tiff := []byte("MM")
	tiff = binary.BigEndian.AppendUint16(tiff, 0x002A)
	tiff = binary.BigEndian.AppendUint32(tiff, tiffHeaderSize)
	tiff = binary.BigEndian.AppendUint16(tiff, 1)
	tiff = binary.BigEndian.AppendUint16(tiff, exifTagOrientation)
	tiff = binary.BigEndian.AppendUint16(tiff, 3)
	tiff = binary.BigEndian.AppendUint32(tiff, 1)
	tiff = binary.BigEndian.AppendUint16(tiff, uint16(orientation))
	tiff = binary.BigEndian.AppendUint16(tiff, 0)
	tiff = binary.BigEndian.AppendUint32(tiff, 0)

	payload := append([]byte(exifIdentifier), tiff...)
	segment := []byte{markerPrefix, markerApplication}
	segment = binary.BigEndian.AppendUint16(segment, uint16(segmentLengthSize+len(payload)))
	segment = append(segment, payload...)

	withExif := append([]byte{}, encoded[:segmentLengthSize]...)
	withExif = append(withExif, segment...)
	return append(withExif, encoded[segmentLengthSize:]...)
}

func TestInspectReportsFormatAndDimensions(t *testing.T) {
	cases := []picture.Format{picture.FormatJPEG, picture.FormatPNG, picture.FormatWebP}

	for _, format := range cases {
		t.Run(format.String(), func(t *testing.T) {
			info, err := New().Inspect(context.Background(), encodedImage(t, format, 64, 32))
			if err != nil {
				t.Fatal(err)
			}
			if info.Format() != format || info.Dimensions().Width() != 64 || info.Dimensions().Height() != 32 {
				t.Errorf("unexpected info %+v", info)
			}
		})
	}
}

func TestInspectRejectsUnknownBytes(t *testing.T) {
	_, err := New().Inspect(context.Background(), []byte("this is not an image at all"))

	if !exception.HasCode(err, exception.CodeImageUnsupportedFormat) {
		t.Errorf("want unsupported format, got %v", err)
	}
}

func TestInspectRejectsTruncatedHeader(t *testing.T) {
	truncated := encodedImage(t, picture.FormatJPEG, 64, 32)[:6]

	_, err := New().Inspect(context.Background(), truncated)

	if !exception.HasCode(err, exception.CodeImageCorrupt) {
		t.Errorf("want corrupt, got %v", err)
	}
}

func TestProcessKeepsFormatAndBuildsThumbnail(t *testing.T) {
	cases := []picture.Format{picture.FormatJPEG, picture.FormatPNG, picture.FormatWebP}

	for _, format := range cases {
		t.Run(format.String(), func(t *testing.T) {
			processor := New()
			data := encodedImage(t, format, 800, 600)
			info, err := processor.Inspect(context.Background(), data)
			if err != nil {
				t.Fatal(err)
			}

			processed, err := processor.Process(context.Background(), data, info, thumbnailSide)
			if err != nil {
				t.Fatal(err)
			}

			finalConfig, finalFormat, err := image.DecodeConfig(bytes.NewReader(processed.Final))
			if err != nil || finalFormat != format.String() || finalConfig.Width != 800 || finalConfig.Height != 600 {
				t.Errorf("final: %+v %q %v", finalConfig, finalFormat, err)
			}

			thumbConfig, thumbFormat, err := image.DecodeConfig(bytes.NewReader(processed.Thumbnail))
			if err != nil || thumbFormat != "webp" || thumbConfig.Width != 400 || thumbConfig.Height != 300 {
				t.Errorf("thumbnail: %+v %q %v", thumbConfig, thumbFormat, err)
			}
		})
	}
}

func TestProcessDoesNotEnlargeSmallImages(t *testing.T) {
	processor := New()
	data := encodedImage(t, picture.FormatPNG, 120, 80)
	info, _ := processor.Inspect(context.Background(), data)

	processed, err := processor.Process(context.Background(), data, info, thumbnailSide)
	if err != nil {
		t.Fatal(err)
	}

	config, _, _ := image.DecodeConfig(bytes.NewReader(processed.Thumbnail))
	if config.Width != 120 || config.Height != 80 {
		t.Errorf("thumbnail must keep size, got %dx%d", config.Width, config.Height)
	}
}

func TestProcessAppliesJPEGOrientation(t *testing.T) {
	processor := New()
	data := jpegWithOrientation(t, 40, 20, orientationRotate90)
	info, err := processor.Inspect(context.Background(), data)
	if err != nil {
		t.Fatal(err)
	}

	processed, err := processor.Process(context.Background(), data, info, thumbnailSide)
	if err != nil {
		t.Fatal(err)
	}

	config, _, _ := image.DecodeConfig(bytes.NewReader(processed.Final))
	if config.Width != 20 || config.Height != 40 {
		t.Errorf("want 20x40 after rotation, got %dx%d", config.Width, config.Height)
	}
}

func TestProcessReportsCorruptBody(t *testing.T) {
	processor := New()
	whole := encodedImage(t, picture.FormatJPEG, 400, 300)
	info, _ := processor.Inspect(context.Background(), whole)

	_, err := processor.Process(context.Background(), whole[:len(whole)/3], info, thumbnailSide)

	if !exception.HasCode(err, exception.CodeImageCorrupt) {
		t.Errorf("want corrupt, got %v", err)
	}
}

func TestProcessStopsWhenCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	data := encodedImage(t, picture.FormatPNG, 10, 10)
	info, _ := New().Inspect(context.Background(), data)

	if _, err := New().Process(ctx, data, info, thumbnailSide); err == nil {
		t.Error("want error when context is canceled")
	}
}

func TestReadOrientation(t *testing.T) {
	cases := map[int]int{1: 1, 3: 3, 6: 6, 8: 8, 0: 1, 9: 1}

	for stored, want := range cases {
		data := jpegWithOrientation(t, 8, 8, stored)
		if got := readOrientation(picture.FormatJPEG, data); got != want {
			t.Errorf("orientation %d: got %d, want %d", stored, got, want)
		}
	}
}

func TestReadOrientationIgnoresOtherFormatsAndMissingExif(t *testing.T) {
	if readOrientation(picture.FormatPNG, jpegWithOrientation(t, 8, 8, 6)) != orientationNormal {
		t.Error("only JPEG carries orientation")
	}
	if readOrientation(picture.FormatJPEG, encodedImage(t, picture.FormatJPEG, 8, 8)) != orientationNormal {
		t.Error("jpeg without exif is upright")
	}
	if readOrientation(picture.FormatJPEG, []byte{0xFF}) != orientationNormal {
		t.Error("garbage is upright")
	}
}

func TestApplyOrientationMapsPixels(t *testing.T) {
	left := color.NRGBA{R: 255, A: 255}
	right := color.NRGBA{B: 255, A: 255}
	source := image.NewNRGBA(image.Rect(0, 0, 2, 1))
	source.SetNRGBA(0, 0, left)
	source.SetNRGBA(1, 0, right)

	cases := map[string]struct {
		orientation   int
		width, height int
		at            map[[2]int]color.NRGBA
	}{
		"flip horizontal": {orientationFlipHorizontal, 2, 1, map[[2]int]color.NRGBA{{0, 0}: right, {1, 0}: left}},
		"rotate 180":      {orientationRotate180, 2, 1, map[[2]int]color.NRGBA{{0, 0}: right, {1, 0}: left}},
		"flip vertical":   {orientationFlipVertical, 2, 1, map[[2]int]color.NRGBA{{0, 0}: left, {1, 0}: right}},
		"transpose":       {orientationTranspose, 1, 2, map[[2]int]color.NRGBA{{0, 0}: left, {0, 1}: right}},
		"rotate 90":       {orientationRotate90, 1, 2, map[[2]int]color.NRGBA{{0, 0}: left, {0, 1}: right}},
		"transverse":      {orientationTransverse, 1, 2, map[[2]int]color.NRGBA{{0, 0}: right, {0, 1}: left}},
		"rotate 270":      {orientationRotate270, 1, 2, map[[2]int]color.NRGBA{{0, 0}: right, {0, 1}: left}},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			result := applyOrientation(source, tc.orientation)

			if result.Bounds().Dx() != tc.width || result.Bounds().Dy() != tc.height {
				t.Fatalf("size %dx%d", result.Bounds().Dx(), result.Bounds().Dy())
			}
			for point, want := range tc.at {
				if got := color.NRGBAModel.Convert(result.At(point[0], point[1])); got != want {
					t.Errorf("pixel %v = %v, want %v", point, got, want)
				}
			}
		})
	}
}

func TestApplyOrientationLeavesUprightImageUntouched(t *testing.T) {
	source := gradient(4, 4)

	if applyOrientation(source, orientationNormal) != image.Image(source) {
		t.Error("upright image must be returned as is")
	}
}

func TestEncodedWebPDecodes(t *testing.T) {
	data := encodedImage(t, picture.FormatWebP, 16, 16)

	if _, err := nativewebp.Decode(bytes.NewReader(data)); err != nil {
		t.Errorf("webp must decode: %v", err)
	}
}

func TestEncodedPNGDecodes(t *testing.T) {
	data := encodedImage(t, picture.FormatPNG, 16, 16)

	if _, err := png.Decode(bytes.NewReader(data)); err != nil {
		t.Errorf("png must decode: %v", err)
	}
}
