package memory

import (
	"context"
	"sort"
	"sync"

	"github.com/google/uuid"

	"rutadeorigen/qr/internal/application/output/ports"
	"rutadeorigen/qr/internal/domain/evidenceview"
)

var _ ports.EvidenceViewRepository = (*EvidenceViewRepository)(nil)

type EvidenceViewRepository struct {
	mu    sync.RWMutex
	views map[uuid.UUID]*evidenceview.EvidenceView
}

func NewEvidenceViewRepository() *EvidenceViewRepository {
	return &EvidenceViewRepository{views: make(map[uuid.UUID]*evidenceview.EvidenceView)}
}

func (r *EvidenceViewRepository) Save(_ context.Context, view *evidenceview.EvidenceView) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.views[view.EvidenceID()]; !exists {
		r.views[view.EvidenceID()] = view
	}
	return nil
}

func (r *EvidenceViewRepository) ListByLotID(_ context.Context, lotID uuid.UUID, limit int) ([]*evidenceview.EvidenceView, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	found := make([]*evidenceview.EvidenceView, 0)
	for _, view := range r.views {
		if view.LotID() == lotID {
			found = append(found, view)
		}
	}

	sort.Slice(found, func(i, j int) bool {
		if !found[i].ProcessedAt().Equal(found[j].ProcessedAt()) {
			return found[i].ProcessedAt().Before(found[j].ProcessedAt())
		}
		return found[i].EvidenceID().String() < found[j].EvidenceID().String()
	})

	if limit >= 0 && len(found) > limit {
		found = found[:limit]
	}
	return found, nil
}
