package rejection

import (
	"maps"

	"rutadeorigen/processor/internal/platform/exception"
	"rutadeorigen/processor/internal/platform/helper"
)

var rejectable = map[exception.Code]bool{
	exception.CodeFileMissing:            true,
	exception.CodeFileTooLarge:           true,
	exception.CodeFileSizeMismatch:       true,
	exception.CodeImageUnsupportedFormat: true,
	exception.CodeImageCorrupt:           true,
	exception.CodeImageInvalidDimensions: true,
	exception.CodeImageTooManyPixels:     true,
}

type Reason struct {
	code   exception.Code
	params map[string]any
}

func Default() Reason {
	return Reason{code: exception.Code(helper.EmptyString), params: map[string]any{}}
}

func FromError(err error) Reason {
	ex, ok := exception.From(err)
	if !ok || !rejectable[ex.Code()] {
		return Default()
	}
	return Reason{code: ex.Code(), params: ex.Params()}
}

func (r Reason) IsDefault() bool { return helper.IsEmpty(string(r.code)) }

func (r Reason) Code() string { return string(r.code) }

func (r Reason) Params() map[string]any { return maps.Clone(r.params) }
