package memory

import (
	"context"
	"sync"

	"github.com/google/uuid"

	"rutadeorigen/traceability/internal/application/output/ports"
	"rutadeorigen/traceability/internal/domain/farm"
)

var _ ports.FarmRepository = (*FarmRepository)(nil)

type FarmRepository struct {
	mu    sync.RWMutex
	farms map[uuid.UUID]*farm.Farm
}

func NewFarmRepository(seed ...*farm.Farm) *FarmRepository {
	r := &FarmRepository{farms: make(map[uuid.UUID]*farm.Farm)}
	for _, f := range seed {
		if !f.IsDefault() {
			r.farms[f.ID()] = f
		}
	}
	return r
}

func (r *FarmRepository) FindByID(_ context.Context, id uuid.UUID) (*farm.Farm, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	f, ok := r.farms[id]
	if !ok {
		return farm.Default(), nil
	}
	return f, nil
}
