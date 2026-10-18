package memory

import (
	"bytes"
	"context"
	"sort"
	"sync"

	"rutadeorigen/traceability/internal/platform/exception"

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

func (r *LotRepository) FindByID(_ context.Context, id uuid.UUID) (*lot.Lot, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	l, ok := r.lots[id]
	if !ok {
		return lot.Default(), nil
	}
	return l, nil
}

func (r *LotRepository) List(_ context.Context, f lot.Filter, after lot.Cursor, limit int) ([]*lot.Lot, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	matches := make([]*lot.Lot, 0, len(r.lots))
	for _, l := range r.lots {
		if f.HasFarm() && l.FarmID() != f.FarmID {
			continue
		}
		if f.HasCustodian() && l.CustodianID() != f.CustodianID {
			continue
		}
		if !after.IsDefault() && !comesAfter(l, after) {
			continue
		}
		matches = append(matches, l)
	}

	sort.Slice(matches, func(i, j int) bool { return newer(matches[i], matches[j]) })
	if len(matches) > limit {
		matches = matches[:limit]
	}
	return matches, nil
}

func newer(a, b *lot.Lot) bool {
	if !a.CreatedAt().Equal(b.CreatedAt()) {
		return a.CreatedAt().After(b.CreatedAt())
	}
	aid, bid := a.ID(), b.ID()
	return bytes.Compare(aid[:], bid[:]) > 0
}

func comesAfter(l *lot.Lot, c lot.Cursor) bool {
	if !l.CreatedAt().Equal(c.CreatedAt) {
		return l.CreatedAt().Before(c.CreatedAt)
	}
	id := l.ID()
	return bytes.Compare(id[:], c.ID[:]) < 0
}
