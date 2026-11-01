package getscanstats

import (
	"time"

	"rutadeorigen/qr/internal/application/input/dto/request"
	"rutadeorigen/qr/internal/application/input/dto/response"
	getscanstatsuc "rutadeorigen/qr/internal/application/usecases/getscanstats"
	"rutadeorigen/qr/internal/domain/scan"
	"rutadeorigen/qr/internal/platform/exception"
	"rutadeorigen/qr/internal/platform/helper"
)

const dayLayout = "2006-01-02"

type Assembler struct{}

func (Assembler) ToQuery(req request.GetScanStats) (getscanstatsuc.Query, error) {
	lotID := helper.ParseUUID(req.LotID)
	if helper.IsDefaultUUID(lotID) {
		return getscanstatsuc.Query{}, exception.New(exception.CodeLotViewNotFound)
	}

	days := scan.DefaultStatsDays
	if !helper.IsEmpty(req.Days) {
		days = helper.ParseInt(req.Days)
	}
	if !scan.IsValidStatsDays(days) {
		return getscanstatsuc.Query{}, exception.New(exception.CodeStatsInvalidDays)
	}

	return getscanstatsuc.Query{LotID: lotID, Days: days}, nil
}

func (Assembler) ToDTO(stats *scan.Stats, days int) response.ScanStats {
	daily := make([]response.DailyScans, 0, len(stats.Daily()))
	for _, d := range stats.Daily() {
		daily = append(daily, response.DailyScans{Date: d.Day.UTC().Format(dayLayout), Count: d.Count})
	}

	return response.ScanStats{
		LotID:       stats.LotID().String(),
		Total:       stats.Total(),
		FirstScanAt: formatTime(stats.FirstScanAt()),
		LastScanAt:  formatTime(stats.LastScanAt()),
		Days:        days,
		Daily:       daily,
	}
}

func formatTime(value time.Time) *string {
	if helper.IsDefaultDate(value) {
		return nil
	}
	formatted := value.UTC().Format(time.RFC3339)
	return &formatted
}
