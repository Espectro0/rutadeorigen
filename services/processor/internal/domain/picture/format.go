package picture

import (
	"strings"

	"rutadeorigen/processor/internal/platform/helper"
)

type Format string

const (
	FormatUnknown Format = "unknown"
	FormatJPEG    Format = "jpeg"
	FormatPNG     Format = "png"
	FormatWebP    Format = "webp"
)

func ParseFormat(name string) Format {
	switch f := Format(strings.ToLower(helper.ApplyTrim(name))); f {
	case FormatJPEG, FormatPNG, FormatWebP:
		return f
	case "jpg":
		return FormatJPEG
	default:
		return FormatUnknown
	}
}

func (f Format) IsValid() bool { return ParseFormat(string(f)) != FormatUnknown }

func (f Format) String() string { return string(f) }

func (f Format) Extension() string {
	switch f {
	case FormatJPEG:
		return "jpg"
	case FormatPNG:
		return "png"
	case FormatWebP:
		return "webp"
	default:
		return helper.EmptyString
	}
}

func (f Format) ContentType() string {
	if !f.IsValid() {
		return helper.EmptyString
	}
	return "image/" + string(f)
}
