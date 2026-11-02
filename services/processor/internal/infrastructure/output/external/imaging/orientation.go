package imaging

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"

	"rutadeorigen/processor/internal/domain/picture"
)

const (
	orientationNormal         = 1
	orientationFlipHorizontal = 2
	orientationRotate180      = 3
	orientationFlipVertical   = 4
	orientationTranspose      = 5
	orientationRotate90       = 6
	orientationTransverse     = 7
	orientationRotate270      = 8
)

const (
	exifTagOrientation = 0x0112
	exifIdentifier     = "Exif\x00\x00"
	ifdEntrySize       = 12
	tiffHeaderSize     = 8
	valueOffsetInEntry = 8

	markerPrefix      = 0xFF
	markerStartImage  = 0xD8
	markerStartScan   = 0xDA
	markerEndImage    = 0xD9
	markerApplication = 0xE1
	segmentHeaderSize = 4
	segmentLengthSize = 2
)

func readOrientation(format picture.Format, data []byte) int {
	if format != picture.FormatJPEG {
		return orientationNormal
	}

	tiff := findExifTIFF(data)
	if len(tiff) == 0 {
		return orientationNormal
	}
	return orientationFromTIFF(tiff)
}

func findExifTIFF(data []byte) []byte {
	if len(data) < segmentLengthSize || data[0] != markerPrefix || data[1] != markerStartImage {
		return nil
	}

	position := segmentLengthSize
	for position+segmentHeaderSize <= len(data) {
		if data[position] != markerPrefix {
			return nil
		}

		marker := data[position+1]
		if marker == markerStartScan || marker == markerEndImage {
			return nil
		}

		length := int(binary.BigEndian.Uint16(data[position+2 : position+segmentHeaderSize]))
		segmentEnd := position + segmentLengthSize + length
		if length < segmentLengthSize || segmentEnd > len(data) {
			return nil
		}

		payload := data[position+segmentHeaderSize : segmentEnd]
		if marker == markerApplication && bytes.HasPrefix(payload, []byte(exifIdentifier)) {
			return payload[len(exifIdentifier):]
		}
		position = segmentEnd
	}
	return nil
}

func orientationFromTIFF(tiff []byte) int {
	if len(tiff) < tiffHeaderSize {
		return orientationNormal
	}

	var order binary.ByteOrder
	switch string(tiff[:2]) {
	case "II":
		order = binary.LittleEndian
	case "MM":
		order = binary.BigEndian
	default:
		return orientationNormal
	}

	directoryOffset := int(order.Uint32(tiff[4:tiffHeaderSize]))
	if directoryOffset < tiffHeaderSize || directoryOffset+segmentLengthSize > len(tiff) {
		return orientationNormal
	}

	entryCount := int(order.Uint16(tiff[directoryOffset:]))
	entriesStart := directoryOffset + segmentLengthSize
	for index := 0; index < entryCount; index++ {
		entry := entriesStart + index*ifdEntrySize
		if entry+ifdEntrySize > len(tiff) {
			return orientationNormal
		}
		if order.Uint16(tiff[entry:]) != exifTagOrientation {
			continue
		}

		value := int(order.Uint16(tiff[entry+valueOffsetInEntry:]))
		if value < orientationNormal || value > orientationRotate270 {
			return orientationNormal
		}
		return value
	}
	return orientationNormal
}

func applyOrientation(img image.Image, orientation int) image.Image {
	if orientation <= orientationNormal || orientation > orientationRotate270 {
		return img
	}

	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	targetWidth, targetHeight := width, height
	if swapsAxes(orientation) {
		targetWidth, targetHeight = height, width
	}

	upright := image.NewNRGBA(image.Rect(0, 0, targetWidth, targetHeight))
	for y := 0; y < targetHeight; y++ {
		for x := 0; x < targetWidth; x++ {
			sourceX, sourceY := sourcePoint(orientation, x, y, width, height)
			upright.Set(x, y, color.NRGBAModel.Convert(img.At(bounds.Min.X+sourceX, bounds.Min.Y+sourceY)))
		}
	}
	return upright
}

func swapsAxes(orientation int) bool {
	return orientation >= orientationTranspose
}

func sourcePoint(orientation, x, y, width, height int) (int, int) {
	switch orientation {
	case orientationFlipHorizontal:
		return width - 1 - x, y
	case orientationRotate180:
		return width - 1 - x, height - 1 - y
	case orientationFlipVertical:
		return x, height - 1 - y
	case orientationTranspose:
		return y, x
	case orientationRotate90:
		return y, height - 1 - x
	case orientationTransverse:
		return width - 1 - y, height - 1 - x
	case orientationRotate270:
		return width - 1 - y, x
	default:
		return x, y
	}
}
