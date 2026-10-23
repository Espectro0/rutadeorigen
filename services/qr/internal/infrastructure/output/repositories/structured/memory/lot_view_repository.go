package memory

import (
	"context"
	"sync"

	"github.com/google/uuid"

	"rutadeorigen/qr/internal/application/output/ports"
	"rutadeorigen/qr/internal/domain/lotview"
)

var _ ports.LotViewRepository = (*LotViewRepository)(nil)

type LotViewRepository struct {
	mu    sync.RWMutex
	views map[uuid.UUID]*lotview.LotView
}

func NewLotViewRepository() *LotViewRepository {
	return &LotViewRepository{views: make(map[uuid.UUID]*lotview.LotView)}
}

func (r *LotViewRepository) FindByID(_ context.Context, id uuid.UUID) (*lotview.LotView, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	view, ok := r.views[id]
	if !ok {
		return lotview.Default(), nil
	}
	return view, nil
}

func (r *LotViewRepository) Save(_ context.Context, view *lotview.LotView) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	current, ok := r.views[view.LotID()]
	if ok && !view.IsNewerThan(current) {
		return nil
	}
	r.views[view.LotID()] = view
	return nil
}
