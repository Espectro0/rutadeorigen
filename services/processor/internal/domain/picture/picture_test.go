package picture

import (
	"testing"

	"rutadeorigen/processor/internal/platform/exception"
)

func TestParseFormat(t *testing.T) {
	cases := map[string]Format{
		"jpeg": FormatJPEG, "JPG": FormatJPEG, " png ": FormatPNG,
		"webp": FormatWebP, "gif": FormatUnknown, "": FormatUnknown,
	}
	for in, want := range cases {
		if got := ParseFormat(in); got != want {
			t.Errorf("ParseFormat(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestFormatExtensionAndContentType(t *testing.T) {
	if FormatJPEG.Extension() != "jpg" || FormatJPEG.ContentType() != "image/jpeg" {
		t.Error("jpeg mapping wrong")
	}
	if FormatUnknown.Extension() != "" || FormatUnknown.ContentType() != "" {
		t.Error("unknown must map to empty")
	}
}

func TestNewDimensions(t *testing.T) {
	if _, err := NewDimensions(0, 10); !exception.HasCode(err, exception.CodeImageInvalidDimensions) {
		t.Errorf("want invalid dimensions, got %v", err)
	}
	d, err := NewDimensions(4000, 3000)
	if err != nil || d.Pixels() != 12_000_000 {
		t.Errorf("unexpected %v %v", d, err)
	}
}

func TestFitWithin(t *testing.T) {
	d, _ := NewDimensions(4000, 3000)
	got := d.FitWithin(400)
	if got.Width() != 400 || got.Height() != 300 {
		t.Errorf("got %dx%d", got.Width(), got.Height())
	}
	small, _ := NewDimensions(200, 100)
	if small.FitWithin(400) != small {
		t.Error("must not enlarge")
	}
	if !DefaultDimensions().FitWithin(400).IsDefault() {
		t.Error("default stays default")
	}
}

func TestLimits(t *testing.T) {
	if _, err := NewLimits(0, 40, 400); err == nil {
		t.Error("expected error")
	}
	l, err := NewLimits(5<<20, 40, 400)
	if err != nil {
		t.Fatal(err)
	}
	if err := l.CheckFileSize(5<<20 + 1); !exception.HasCode(err, exception.CodeFileTooLarge) {
		t.Errorf("want too large, got %v", err)
	}
	if err := l.CheckFileSize(5 << 20); err != nil {
		t.Error(err)
	}
	justUnderLimit, _ := NewDimensions(6000, 6666)
	if err := l.CheckDimensions(justUnderLimit); err != nil {
		t.Error(err)
	}
	overLimit, _ := NewDimensions(7000, 6000)
	if err := l.CheckDimensions(overLimit); !exception.HasCode(err, exception.CodeImageTooManyPixels) {
		t.Errorf("want too many pixels, got %v", err)
	}
	if l.MaxMegapixels() != 40 {
		t.Error("megapixels")
	}
}

func TestNewInfo(t *testing.T) {
	if _, err := NewInfo("gif", 10, 10); !exception.HasCode(err, exception.CodeImageUnsupportedFormat) {
		t.Errorf("got %v", err)
	}
	if _, err := NewInfo("png", 0, 10); !exception.HasCode(err, exception.CodeImageInvalidDimensions) {
		t.Errorf("got %v", err)
	}
	i, err := NewInfo("jpeg", 10, 20)
	if err != nil || i.IsDefault() || i.Format() != FormatJPEG {
		t.Errorf("got %v %v", i, err)
	}
	if !DefaultInfo().IsDefault() {
		t.Error("default info")
	}
}
