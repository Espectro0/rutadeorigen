package memory

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"

	"rutadeorigen/qr/internal/application/output/ports"
	"rutadeorigen/qr/internal/domain/scan"
	"rutadeorigen/qr/internal/platform/helper"
)

var _ ports.ScanRepository = (*ScanRepository)(nil)

type ScanRepository struct {
	mu    sync.RWMutex
	scans []*scan.Scan
}

func NewScanRepository() *ScanRepository {
	return &ScanRepository{}
}

func (r *ScanRepository) Save(_ context.Context, s *scan.Scan) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.scans = append(r.scans, s)
	return nil
}

func (r *ScanRepository) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.scans)
}

func (r *ScanRepository) StatsByLot(_ context.Context, lotID uuid.UUID, since time.Time) (*scan.Stats, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var (
		total       int
		first, last time.Time
		perDay      = make(map[time.Time]int)
	)
	for _, s := range r.scans {
		if s.LotID() != lotID {
			continue
		}

		at := s.ScannedAt().UTC()
		total++
		if first.IsZero() || at.Before(first) {
			first = at
		}
		if at.After(last) {
			last = at
		}
		if !at.Before(since) {
			perDay[helper.DateUTC(at)]++
		}
	}

	daily := make([]scan.DailyCount, 0, len(perDay))
	for day, count := range perDay {
		daily = append(daily, scan.DailyCount{Day: day, Count: count})
	}
	sort.Slice(daily, func(i, j int) bool { return daily[i].Day.Before(daily[j].Day) })

	return scan.NewStats(lotID, total, first, last, daily), nil
}
