package qrcode

type Size int

const (
	SizeUnknown Size = 0
	SizeSmall   Size = 256
	SizeMedium  Size = 512
	SizeLarge   Size = 1024
)

const DefaultSize = SizeMedium

func ParseSize(pixels int) Size {
	switch v := Size(pixels); v {
	case SizeSmall, SizeMedium, SizeLarge:
		return v
	default:
		return SizeUnknown
	}
}

func (v Size) IsValid() bool {
	return ParseSize(int(v)) != SizeUnknown
}

func (v Size) Pixels() int {
	return int(v)
}
