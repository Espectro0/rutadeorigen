package picture

import "rutadeorigen/processor/internal/platform/exception"

type Dimensions struct {
	width  int
	height int
}

func DefaultDimensions() Dimensions { return Dimensions{} }

func NewDimensions(width, height int) (Dimensions, error) {
	if width <= 0 || height <= 0 {
		return DefaultDimensions(), exception.New(exception.CodeImageInvalidDimensions)
	}
	return Dimensions{width: width, height: height}, nil
}

func (d Dimensions) Width() int      { return d.width }
func (d Dimensions) Height() int     { return d.height }
func (d Dimensions) IsDefault() bool { return d.width <= 0 || d.height <= 0 }

func (d Dimensions) Pixels() int64 { return int64(d.width) * int64(d.height) }

func (d Dimensions) FitWithin(side int) Dimensions {
	if d.IsDefault() || side <= 0 {
		return DefaultDimensions()
	}

	longest := max(d.width, d.height)
	if longest <= side {
		return d
	}

	scale := func(value int) int {
		scaled := int((int64(value)*int64(side) + int64(longest)/2) / int64(longest))
		return max(scaled, 1)
	}
	return Dimensions{width: scale(d.width), height: scale(d.height)}
}
