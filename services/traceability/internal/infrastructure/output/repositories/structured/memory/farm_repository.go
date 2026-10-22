package memory

import (
	"bytes"
	"context"
	"sort"
	"sync"

	"github.com/google/uuid"

	"rutadeorigen/traceability/internal/application/output/ports"
	"rutadeorigen/traceability/internal/domain/farm"
	"rutadeorigen/traceability/internal/platform/exception"
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

func (r *FarmRepository) Save(_ context.Context, f *farm.Farm) error {
	if f.IsDefault() {
		return exception.New(exception.CodeInternal).WithTechnical("cannot save a default farm")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.farms[f.ID()] = f
	return nil
}

func (r *FarmRepository) ListByProducer(_ context.Context, producerID uuid.UUID) ([]*farm.Farm, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	farms := make([]*farm.Farm, 0)
	for _, f := range r.farms {
		if f.ProducerID() == producerID {
			farms = append(farms, f)
		}
	}

	sort.Slice(farms, func(i, j int) bool {
		if !farms[i].CreatedAt().Equal(farms[j].CreatedAt()) {
			return farms[i].CreatedAt().After(farms[j].CreatedAt())
		}
		a, b := farms[i].ID(), farms[j].ID()
		return bytes.Compare(a[:], b[:]) > 0
	})
	return farms, nil
}
