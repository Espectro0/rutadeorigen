package helper_test

import (
	"testing"

	"rutadeorigen/traceability/internal/platform/helper"
)

func TestDefaultString(t *testing.T) {
	if got := helper.DefaultString(); got != helper.EmptyString {
		t.Errorf("got %q, want empty string", got)
	}
}

func TestTextOrDefault(t *testing.T) {
	value := "Castillo"

	if got := helper.TextOrDefault(nil); got != helper.EmptyString {
		t.Errorf("nil: got %q, want empty string", got)
	}
	if got := helper.TextOrDefault(&value); got != "Castillo" {
		t.Errorf("got %q, want %q", got, "Castillo")
	}
}

func TestTextIfEmpty(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{"empty string returns default", "", "default"},
		{"only spaces returns default", "   ", "default"},
		{"tabs and newlines return default", "\t\n", "default"},
		{"text returns itself", "Caturra", "Caturra"},
		{"text with spaces is returned untouched", "  Caturra  ", "  Caturra  "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := helper.TextIfEmpty(tt.value, "default"); got != tt.want {
				t.Errorf("TextIfEmpty(%q) = %q, want %q", tt.value, got, tt.want)
			}
		})
	}
}

func TestApplyTrim(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{"empty string", "", ""},
		{"only spaces", "   ", ""},
		{"leading and trailing spaces", "  Finca La Esperanza  ", "Finca La Esperanza"},
		{"tabs and newlines", "\tCafé\n", "Café"},
		{"inner spaces are kept", "La  Esperanza", "La  Esperanza"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := helper.ApplyTrim(tt.value); got != tt.want {
				t.Errorf("ApplyTrim(%q) = %q, want %q", tt.value, got, tt.want)
			}
		})
	}
}

func TestIsEmpty(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{"empty string", "", true},
		{"only spaces", "   ", true},
		{"tabs and newlines", "\t\n", true},
		{"normal text", "Finca La Esperanza", false},
		{"text surrounded by spaces", "  Café  ", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := helper.IsEmpty(tt.value); got != tt.want {
				t.Errorf("IsEmpty(%q) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

func TestLengthIsValid(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{"exactly min length", "abc", true},
		{"exactly max length", "abcde", true},
		{"below min length", "ab", false},
		{"above max length", "abcdef", false},
		{"spaces are trimmed before counting", "  ab  ", false},
		{"accents count as one character", "Café", true},
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := helper.LengthIsValid(tt.value, 3, 5); got != tt.want {
				t.Errorf("LengthIsValid(%q, 3, 5) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}
