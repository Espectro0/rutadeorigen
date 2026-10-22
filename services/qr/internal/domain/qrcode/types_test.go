package qrcode_test

import (
	"testing"

	"rutadeorigen/qr/internal/domain/qrcode"
)

func TestParseErrorCorrection(t *testing.T) {
	tests := []struct {
		value string
		want  qrcode.ErrorCorrection
	}{
		{"L", qrcode.ErrorCorrectionLow},
		{"m", qrcode.ErrorCorrectionMedium},
		{" q ", qrcode.ErrorCorrectionQuartile},
		{"H", qrcode.ErrorCorrectionHigh},
		{"", qrcode.ErrorCorrectionUnknown},
		{"X", qrcode.ErrorCorrectionUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			if got := qrcode.ParseErrorCorrection(tt.value); got != tt.want {
				t.Errorf("ParseErrorCorrection(%q) = %q, want %q", tt.value, got, tt.want)
			}
		})
	}

	if !qrcode.DefaultErrorCorrection.IsValid() || qrcode.ErrorCorrectionUnknown.IsValid() {
		t.Error("ErrorCorrection.IsValid failed")
	}
	if qrcode.ErrorCorrectionQuartile.String() != "Q" {
		t.Error("ErrorCorrection.String failed")
	}
}

func TestParseFormat(t *testing.T) {
	tests := []struct {
		value string
		want  qrcode.Format
	}{
		{"png", qrcode.FormatPNG},
		{"SVG", qrcode.FormatSVG},
		{" png ", qrcode.FormatPNG},
		{"", qrcode.FormatUnknown},
		{"jpg", qrcode.FormatUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			if got := qrcode.ParseFormat(tt.value); got != tt.want {
				t.Errorf("ParseFormat(%q) = %q, want %q", tt.value, got, tt.want)
			}
		})
	}
}

func TestFormatContentTypeAndExtension(t *testing.T) {
	if qrcode.FormatPNG.ContentType() != "image/png" || qrcode.FormatPNG.Extension() != "png" {
		t.Error("unexpected png values")
	}
	if qrcode.FormatSVG.ContentType() != "image/svg+xml" || qrcode.FormatSVG.Extension() != "svg" {
		t.Error("unexpected svg values")
	}
	if qrcode.FormatUnknown.ContentType() != "" || qrcode.FormatUnknown.Extension() != "" {
		t.Error("unknown format must have empty content type and extension")
	}
	if !qrcode.DefaultFormat.IsValid() || qrcode.FormatUnknown.IsValid() || qrcode.FormatSVG.String() != "svg" {
		t.Error("Format.IsValid or String failed")
	}
}

func TestParseSize(t *testing.T) {
	tests := []struct {
		pixels int
		want   qrcode.Size
	}{
		{256, qrcode.SizeSmall},
		{512, qrcode.SizeMedium},
		{1024, qrcode.SizeLarge},
		{0, qrcode.SizeUnknown},
		{-256, qrcode.SizeUnknown},
		{300, qrcode.SizeUnknown},
		{2048, qrcode.SizeUnknown},
	}

	for _, tt := range tests {
		if got := qrcode.ParseSize(tt.pixels); got != tt.want {
			t.Errorf("ParseSize(%d) = %d, want %d", tt.pixels, got, tt.want)
		}
	}

	if !qrcode.DefaultSize.IsValid() || qrcode.SizeUnknown.IsValid() {
		t.Error("Size.IsValid failed")
	}
	if qrcode.SizeLarge.Pixels() != 1024 {
		t.Error("Size.Pixels failed")
	}
}
