package memory

import (
	"context"
	"sync"

	"github.com/google/uuid"

	"rutadeorigen/qr/internal/application/output/ports"
	"rutadeorigen/qr/internal/domain/lotview"
)

var _ ports.LotViewCache = (*LotViewCache)(nil)

type LotViewCache struct {
	mu    sync.RWMutex
	views map[uuid.UUID]*lotview.LotView
}

func NewLotViewCache() *LotViewCache {
	return &LotViewCache{views: make(map[uuid.UUID]*lotview.LotView)}
}

func (c *LotViewCache) Get(_ context.Context, id uuid.UUID) (*lotview.LotView, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	view, ok := c.views[id]
	if !ok {
		return lotview.Default(), nil
	}
	return view, nil
}

func (c *LotViewCache) Set(_ context.Context, view *lotview.LotView) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.views[view.LotID()] = view
	return nil
}

func (c *LotViewCache) Delete(_ context.Context, id uuid.UUID) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.views, id)
	return nil
}
