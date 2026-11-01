package scan

import (
	"time"

	"github.com/google/uuid"

	"rutadeorigen/qr/internal/platform/helper"
)

const (
	DefaultStatsDays = 30
	MaxStatsDays     = 365
)

func IsValidStatsDays(days int) bool {
	return helper.IsInRange(days, 1, MaxStatsDays)
}

type DailyCount struct {
	Day   time.Time
	Count int
}

type Stats struct {
	lotID       uuid.UUID
	total       int
	firstScanAt time.Time
	lastScanAt  time.Time
	daily       []DailyCount
}

func NewStats(lotID uuid.UUID, total int, firstScanAt, lastScanAt time.Time, daily []DailyCount) *Stats {
	if total < 0 {
		total = 0
	}

	return &Stats{
		lotID:       lotID,
		total:       total,
		firstScanAt: firstScanAt.UTC(),
		lastScanAt:  lastScanAt.UTC(),
		daily:       append([]DailyCount(nil), daily...),
	}
}

func (s *Stats) LotID() uuid.UUID       { return s.lotID }
func (s *Stats) Total() int             { return s.total }
func (s *Stats) FirstScanAt() time.Time { return s.firstScanAt }
func (s *Stats) LastScanAt() time.Time  { return s.lastScanAt }
func (s *Stats) HasScans() bool         { return s.total > 0 }
func (s *Stats) Daily() []DailyCount {
	return append(make([]DailyCount, 0, len(s.daily)), s.daily...)
}
