package memory

import (
	"context"
	"rutadeorigen/traceability/internal/platform/exception"
	"sync"

	"github.com/google/uuid"

	"rutadeorigen/traceability/internal/application/output/ports"
	"rutadeorigen/traceability/internal/domain/lot"
)

var _ ports.LotRepository = (*LotRepository)(nil)

type LotRepository struct {
	mu   sync.RWMutex
	lots map[uuid.UUID]*lot.Lot
}

func NewLotRepository(seed ...*lot.Lot) *LotRepository {
	r := &LotRepository{lots: make(map[uuid.UUID]*lot.Lot)}
	for _, l := range seed {
		if !l.IsDefault() {
			r.lots[l.ID()] = l
		}
	}
	return r
}

func (r *LotRepository) ExistsByCode(_ context.Context, farmID uuid.UUID, code string) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, l := range r.lots {
		if l.FarmID() == farmID && l.Code() == code {
			return true, nil
		}
	}
	return false, nil
}

func (r *LotRepository) Save(_ context.Context, l *lot.Lot) error {
	if l.IsDefault() {
		return exception.New(exception.CodeInternal).WithTechnical("cannot save a default lot")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.lots[l.ID()] = l
	return nil
}
