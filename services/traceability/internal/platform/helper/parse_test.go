package helper_test

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"rutadeorigen/traceability/internal/platform/helper"
)

func TestParseUUID(t *testing.T) {
	valid := uuid.MustParse("3f2504e0-4f89-11d3-9a0c-0305e82c3301")

	tests := []struct {
		name  string
		value string
		want  uuid.UUID
	}{
		{"valid UUID", "3f2504e0-4f89-11d3-9a0c-0305e82c3301", valid},
		{"valid UUID with spaces", "  3f2504e0-4f89-11d3-9a0c-0305e82c3301  ", valid},
		{"empty string", "", helper.DefaultUUID()},
		{"only spaces", "   ", helper.DefaultUUID()},
		{"invalid text", "not-a-uuid", helper.DefaultUUID()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := helper.ParseUUID(tt.value); got != tt.want {
				t.Errorf("ParseUUID(%q) = %s, want %s", tt.value, got, tt.want)
			}
		})
	}
}

func TestParseInt(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  int
	}{
		{"positive number", "1650", 1650},
		{"negative number", "-20", -20},
		{"number with spaces", " 12 ", 12},
		{"empty string", "", helper.DefaultInt()},
		{"invalid text", "abc", helper.DefaultInt()},
		{"decimal number", "12.5", helper.DefaultInt()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := helper.ParseInt(tt.value); got != tt.want {
				t.Errorf("ParseInt(%q) = %d, want %d", tt.value, got, tt.want)
			}
		})
	}
}

func TestParseFloat(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  float64
	}{
		{"decimal number", "70.5", 70.5},
		{"integer number", "70", 70},
		{"number with spaces", " 1.25 ", 1.25},
		{"empty string", "", helper.DefaultFloat()},
		{"invalid text", "abc", helper.DefaultFloat()},
		{"comma as decimal separator", "70,5", helper.DefaultFloat()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := helper.ParseFloat(tt.value); got != tt.want {
				t.Errorf("ParseFloat(%q) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

func TestParseDate(t *testing.T) {
	want := time.Date(2026, time.October, 9, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name  string
		value string
		want  time.Time
	}{
		{"valid date", "2026-10-09", want},
		{"valid date with spaces", " 2026-10-09 ", want},
		{"empty string", "", helper.DefaultDate()},
		{"wrong format", "09/10/2026", helper.DefaultDate()},
		{"impossible date", "2026-02-30", helper.DefaultDate()},
		{"date with time", "2026-10-09T10:00:00Z", helper.DefaultDate()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := helper.ParseDate(tt.value); !got.Equal(tt.want) {
				t.Errorf("ParseDate(%q) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

func TestParseDateTime(t *testing.T) {
	bogota := time.FixedZone("COT", -5*60*60)
	want := time.Date(2026, time.October, 9, 18, 43, 0, 0, bogota)

	tests := []struct {
		name  string
		value string
		want  time.Time
	}{
		{"valid date time with offset", "2026-10-09T18:43:00-05:00", want},
		{"same instant in UTC", "2026-10-09T23:43:00Z", want},
		{"empty string", "", helper.DefaultDate()},
		{"date without time", "2026-10-09", helper.DefaultDate()},
		{"invalid text", "yesterday", helper.DefaultDate()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := helper.ParseDateTime(tt.value); !got.Equal(tt.want) {
				t.Errorf("ParseDateTime(%q) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}
