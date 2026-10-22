package qrcode

import (
	"strings"

	"rutadeorigen/qr/internal/platform/exception"
	"rutadeorigen/qr/internal/platform/helper"
)

const (
	filenamePrefix   = "qr-"
	filenameFallback = "lot"
)

type Image struct {
	data     []byte
	format   Format
	filename string
}

var defaultImage = Image{
	data:     []byte{},
	format:   FormatUnknown,
	filename: helper.EmptyString,
}

func DefaultImage() *Image {
	i := defaultImage
	i.data = []byte{}
	return &i
}

func NewImage(data []byte, format Format, lotCode string) (*Image, error) {
	if len(data) == 0 {
		return DefaultImage(), exception.New(exception.CodeQRCodeInvalidContent)
	}
	if !format.IsValid() {
		return DefaultImage(), exception.New(exception.CodeQRCodeInvalidFormat)
	}

	return &Image{
		data:     data,
		format:   format,
		filename: filenamePrefix + safeName(lotCode) + "." + format.Extension(),
	}, nil
}

func safeName(lotCode string) string {
	name := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			return r
		default:
			return '-'
		}
	}, helper.ApplyTrim(lotCode))

	name = strings.Trim(name, "-")
	if name == helper.EmptyString {
		return filenameFallback
	}
	return name
}

func (i *Image) Data() []byte        { return i.data }
func (i *Image) Format() Format      { return i.format }
func (i *Image) ContentType() string { return i.format.ContentType() }
func (i *Image) Filename() string    { return i.filename }

func (i *Image) IsDefault() bool {
	return i == nil || len(i.data) == 0
}
