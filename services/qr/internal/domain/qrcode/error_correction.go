package qrcode

import (
	"strings"

	"rutadeorigen/qr/internal/platform/helper"
)

type ErrorCorrection string

const (
	ErrorCorrectionUnknown  ErrorCorrection = "unknown"
	ErrorCorrectionLow      ErrorCorrection = "L"
	ErrorCorrectionMedium   ErrorCorrection = "M"
	ErrorCorrectionQuartile ErrorCorrection = "Q"
	ErrorCorrectionHigh     ErrorCorrection = "H"
)

const DefaultErrorCorrection = ErrorCorrectionQuartile

func ParseErrorCorrection(value string) ErrorCorrection {
	switch v := ErrorCorrection(strings.ToUpper(helper.ApplyTrim(value))); v {
	case ErrorCorrectionLow, ErrorCorrectionMedium, ErrorCorrectionQuartile, ErrorCorrectionHigh:
		return v
	default:
		return ErrorCorrectionUnknown
	}
}

func (v ErrorCorrection) IsValid() bool {
	return ParseErrorCorrection(string(v)) != ErrorCorrectionUnknown
}

func (v ErrorCorrection) String() string {
	return string(v)
}
