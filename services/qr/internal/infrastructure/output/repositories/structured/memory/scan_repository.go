package memory

import (
	"context"
	"sync"

	"rutadeorigen/qr/internal/application/output/ports"
	"rutadeorigen/qr/internal/domain/scan"
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
