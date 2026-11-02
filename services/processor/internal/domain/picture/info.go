package picture

import "rutadeorigen/processor/internal/platform/exception"

type Info struct {
	format     Format
	dimensions Dimensions
}

func DefaultInfo() Info {
	return Info{format: FormatUnknown, dimensions: DefaultDimensions()}
}

func NewInfo(formatName string, width, height int) (Info, error) {
	format := ParseFormat(formatName)
	if !format.IsValid() {
		return DefaultInfo(), exception.New(exception.CodeImageUnsupportedFormat)
	}

	dimensions, err := NewDimensions(width, height)
	if err != nil {
		return DefaultInfo(), err
	}
	return Info{format: format, dimensions: dimensions}, nil
}

func (i Info) Format() Format         { return i.format }
func (i Info) Dimensions() Dimensions { return i.dimensions }
func (i Info) IsDefault() bool        { return !i.format.IsValid() || i.dimensions.IsDefault() }
