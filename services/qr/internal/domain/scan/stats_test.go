package scan_test

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"rutadeorigen/qr/internal/domain/scan"
)

func TestNewStatsKeepsTheData(t *testing.T) {
	id := uuid.New()
	first := time.Date(2026, 10, 1, 8, 0, 0, 0, time.FixedZone("x", -5*3600))
	last := time.Date(2026, 10, 3, 9, 0, 0, 0, time.UTC)
	daily := []scan.DailyCount{{Day: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC), Count: 2}}

	got := scan.NewStats(id, 3, first, last, daily)

	if got.LotID() != id || got.Total() != 3 || !got.HasScans() {
		t.Errorf("unexpected stats %+v", got)
	}
	if !got.FirstScanAt().Equal(first) || got.FirstScanAt().Location() != time.UTC || !got.LastScanAt().Equal(last) {
		t.Errorf("unexpected dates %v %v", got.FirstScanAt(), got.LastScanAt())
	}
	if len(got.Daily()) != 1 || got.Daily()[0].Count != 2 {
		t.Errorf("unexpected daily %+v", got.Daily())
	}
}

func TestStatsWithoutScans(t *testing.T) {
	got := scan.NewStats(uuid.New(), -4, time.Time{}, time.Time{}, nil)

	if got.Total() != 0 || got.HasScans() || len(got.Daily()) != 0 {
		t.Errorf("a negative total must become zero, got %+v", got)
	}
}

func TestStatsDoesNotShareTheDailySlice(t *testing.T) {
	daily := []scan.DailyCount{{Count: 1}}
	got := scan.NewStats(uuid.New(), 1, time.Time{}, time.Time{}, daily)

	daily[0].Count = 99
	got.Daily()[0].Count = 98

	if got.Daily()[0].Count != 1 {
		t.Error("the stats must keep their own copy of the daily counts")
	}
}

func TestIsValidStatsDays(t *testing.T) {
	for days, want := range map[int]bool{-1: false, 0: false, 1: true, scan.DefaultStatsDays: true, scan.MaxStatsDays: true, scan.MaxStatsDays + 1: false} {
		if got := scan.IsValidStatsDays(days); got != want {
			t.Errorf("IsValidStatsDays(%d) = %v, want %v", days, got, want)
		}
	}
}
