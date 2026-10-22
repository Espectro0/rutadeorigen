package qrcode_test

import (
	"testing"

	"rutadeorigen/qr/internal/domain/qrcode"
	"rutadeorigen/qr/internal/platform/exception"
)

func TestNewImageValid(t *testing.T) {
	data := []byte{0x89, 0x50, 0x4e, 0x47}

	i, err := qrcode.NewImage(data, qrcode.FormatPNG, "LT-001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if string(i.Data()) != string(data) || i.Format() != qrcode.FormatPNG {
		t.Error("unexpected data or format")
	}
	if i.ContentType() != "image/png" || i.Filename() != "qr-LT-001.png" {
		t.Errorf("unexpected content type %q or filename %q", i.ContentType(), i.Filename())
	}
	if i.IsDefault() {
		t.Error("a valid image must not be default")
	}
}

func TestNewImageFilename(t *testing.T) {
	tests := []struct {
		name    string
		lotCode string
		format  qrcode.Format
		want    string
	}{
		{"plain code", "LT-001", qrcode.FormatSVG, "qr-LT-001.svg"},
		{"spaces and symbols", "  LT 001/../x  ", qrcode.FormatPNG, "qr-LT-001----x.png"},
		{"accents", "Café-01", qrcode.FormatPNG, "qr-Caf--01.png"},
		{"only symbols", "///", qrcode.FormatPNG, "qr-lot.png"},
		{"empty code", "", qrcode.FormatSVG, "qr-lot.svg"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i, err := qrcode.NewImage([]byte{1}, tt.format, tt.lotCode)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if i.Filename() != tt.want {
				t.Errorf("Filename: got %q, want %q", i.Filename(), tt.want)
			}
		})
	}
}

func TestNewImageFilenameNeverContainsUnsafeCharacters(t *testing.T) {
	i, err := qrcode.NewImage([]byte{1}, qrcode.FormatPNG, `..\..\evil "name"/;`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, r := range i.Filename() {
		safe := r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' || r == '.'
		if !safe {
			t.Errorf("filename %q has the unsafe character %q", i.Filename(), r)
		}
	}
}

func TestNewImageInvalid(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		format   qrcode.Format
		wantCode exception.Code
	}{
		{"nil data", nil, qrcode.FormatPNG, exception.CodeQRCodeInvalidContent},
		{"empty data", []byte{}, qrcode.FormatPNG, exception.CodeQRCodeInvalidContent},
		{"unknown format", []byte{1}, qrcode.FormatUnknown, exception.CodeQRCodeInvalidFormat},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i, err := qrcode.NewImage(tt.data, tt.format, "LT-001")
			if !exception.HasCode(err, tt.wantCode) {
				t.Errorf("got %v, want code %q", err, tt.wantCode)
			}
			if !i.IsDefault() {
				t.Error("expected default image on error")
			}
		})
	}
}

func TestDefaultImage(t *testing.T) {
	i := qrcode.DefaultImage()

	if !i.IsDefault() || i.Data() == nil || len(i.Data()) != 0 {
		t.Error("Default image must be default with empty non nil data")
	}
	if i.Format() != qrcode.FormatUnknown || i.ContentType() != "" || i.Filename() != "" {
		t.Error("unexpected default values")
	}
	if qrcode.DefaultImage() == i {
		t.Error("DefaultImage must return a new instance each time")
	}
}

func TestImageIsDefaultOnNil(t *testing.T) {
	var i *qrcode.Image

	if !i.IsDefault() {
		t.Error("a nil image must be default")
	}
}
