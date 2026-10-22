package qrcode

import (
	"strings"

	"rutadeorigen/qr/internal/platform/helper"
)

type Format string

const (
	FormatUnknown Format = "unknown"
	FormatPNG     Format = "png"
	FormatSVG     Format = "svg"
)

const DefaultFormat = FormatPNG

func ParseFormat(value string) Format {
	switch v := Format(strings.ToLower(helper.ApplyTrim(value))); v {
	case FormatPNG, FormatSVG:
		return v
	default:
		return FormatUnknown
	}
}

func (v Format) IsValid() bool {
	return ParseFormat(string(v)) != FormatUnknown
}

func (v Format) ContentType() string {
	switch v {
	case FormatPNG:
		return "image/png"
	case FormatSVG:
		return "image/svg+xml"
	default:
		return helper.EmptyString
	}
}

func (v Format) Extension() string {
	if !v.IsValid() {
		return helper.EmptyString
	}
	return string(v)
}

func (v Format) String() string {
	return string(v)
}
