package getscanstats_test

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"rutadeorigen/qr/internal/application/input/adapters/getscanstats"
	"rutadeorigen/qr/internal/application/input/dto/request"
	"rutadeorigen/qr/internal/domain/scan"
	"rutadeorigen/qr/internal/platform/exception"
)

func TestToQueryParsesTheRequest(t *testing.T) {
	id := uuid.New()

	got, err := getscanstats.Assembler{}.ToQuery(request.GetScanStats{LotID: id.String(), Days: " 7 "})

	if err != nil || got.LotID != id || got.Days != 7 {
		t.Errorf("got %+v and %v", got, err)
	}
}

func TestToQueryUsesTheDefaultDays(t *testing.T) {
	got, err := getscanstats.Assembler{}.ToQuery(request.GetScanStats{LotID: uuid.NewString()})

	if err != nil || got.Days != scan.DefaultStatsDays {
		t.Errorf("got %+v and %v", got, err)
	}
}

func TestToQueryRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name string
		req  request.GetScanStats
		code exception.Code
	}{
		{"invalid lot id", request.GetScanStats{LotID: "abc"}, exception.CodeLotViewNotFound},
		{"not a number", request.GetScanStats{LotID: uuid.NewString(), Days: "abc"}, exception.CodeStatsInvalidDays},
		{"zero", request.GetScanStats{LotID: uuid.NewString(), Days: "0"}, exception.CodeStatsInvalidDays},
		{"negative", request.GetScanStats{LotID: uuid.NewString(), Days: "-3"}, exception.CodeStatsInvalidDays},
		{"too many", request.GetScanStats{LotID: uuid.NewString(), Days: "366"}, exception.CodeStatsInvalidDays},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := getscanstats.Assembler{}.ToQuery(tt.req)

			if !exception.HasCode(err, tt.code) {
				t.Errorf("got error %v, want %s", err, tt.code)
			}
		})
	}
}

func TestToDTOMapsTheStats(t *testing.T) {
	id := uuid.New()
	first := time.Date(2026, 10, 1, 8, 30, 0, 0, time.UTC)
	last := time.Date(2026, 10, 3, 9, 0, 0, 0, time.UTC)
	stats := scan.NewStats(id, 5, first, last, []scan.DailyCount{
		{Day: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC), Count: 2},
		{Day: time.Date(2026, 10, 3, 0, 0, 0, 0, time.UTC), Count: 3},
	})

	got := getscanstats.Assembler{}.ToDTO(stats, 30)

	if got.LotID != id.String() || got.Total != 5 || got.Days != 30 {
		t.Errorf("unexpected dto %+v", got)
	}
	if got.FirstScanAt == nil || *got.FirstScanAt != "2026-10-01T08:30:00Z" || got.LastScanAt == nil || *got.LastScanAt != "2026-10-03T09:00:00Z" {
		t.Errorf("unexpected dates %v %v", got.FirstScanAt, got.LastScanAt)
	}
	if len(got.Daily) != 2 || got.Daily[0].Date != "2026-10-01" || got.Daily[0].Count != 2 || got.Daily[1].Date != "2026-10-03" {
		t.Errorf("unexpected daily %+v", got.Daily)
	}
}

func TestToDTOWithoutScansHasNullDatesAndAnEmptyList(t *testing.T) {
	stats := scan.NewStats(uuid.New(), 0, time.Time{}, time.Time{}, nil)

	got := getscanstats.Assembler{}.ToDTO(stats, 7)

	if got.FirstScanAt != nil || got.LastScanAt != nil || got.Daily == nil || len(got.Daily) != 0 {
		t.Errorf("unexpected dto %+v", got)
	}
}
