package helper_test

import (
	"testing"
	"time"

	"rutadeorigen/qr/internal/platform/helper"
)

func TestDefaultDate(t *testing.T) {
	if got := helper.DefaultDate(); !got.IsZero() {
		t.Errorf("got %v, want zero time", got)
	}
}

func TestDateOrDefault(t *testing.T) {
	value := time.Date(2026, time.October, 9, 0, 0, 0, 0, time.UTC)

	if got := helper.DateOrDefault(nil); !helper.IsDefaultDate(got) {
		t.Errorf("nil: got %v, want default date", got)
	}
	if got := helper.DateOrDefault(&value); !got.Equal(value) {
		t.Errorf("got %v, want %v", got, value)
	}
}

func TestIsDefaultDate(t *testing.T) {
	if !helper.IsDefaultDate(time.Time{}) {
		t.Error("expected zero time to be the default date")
	}
	if helper.IsDefaultDate(time.Now()) {
		t.Error("expected current time not to be the default date")
	}
}

func TestDateUTC(t *testing.T) {
	bogota := time.FixedZone("COT", -5*60*60)

	tests := []struct {
		name  string
		value time.Time
		want  time.Time
	}{
		{"UTC midnight stays the same", time.Date(2026, time.October, 13, 0, 0, 0, 0, time.UTC), time.Date(2026, time.October, 13, 0, 0, 0, 0, time.UTC)},
		{"time of day is removed", time.Date(2026, time.October, 13, 15, 42, 10, 5, time.UTC), time.Date(2026, time.October, 13, 0, 0, 0, 0, time.UTC)},
		{"offset morning keeps the day", time.Date(2026, time.October, 13, 10, 0, 0, 0, bogota), time.Date(2026, time.October, 13, 0, 0, 0, 0, time.UTC)},
		{"offset night is already the next UTC day", time.Date(2026, time.October, 13, 20, 0, 0, 0, bogota), time.Date(2026, time.October, 14, 0, 0, 0, 0, time.UTC)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := helper.DateUTC(tt.value)
			if !got.Equal(tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
			if got.Location() != time.UTC {
				t.Errorf("location = %v, want UTC", got.Location())
			}
		})
	}
}

func TestIsFutureDate(t *testing.T) {
	today := helper.DateUTC(time.Now())

	tests := []struct {
		name  string
		value time.Time
		want  bool
	}{
		{"yesterday", today.AddDate(0, 0, -1), false},
		{"today at midnight", today, false},
		{"today at the last second", today.Add(24*time.Hour - time.Second), false},
		{"tomorrow", today.AddDate(0, 0, 1), true},
		{"now in a +14 zone is still today in UTC", time.Now().In(time.FixedZone("LINT", 14*60*60)), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := helper.IsFutureDate(tt.value); got != tt.want {
				t.Errorf("IsFutureDate(%v) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}
