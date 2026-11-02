package memory

import (
	"bytes"
	"context"
	"sort"
	"sync"

	"github.com/google/uuid"

	"rutadeorigen/traceability/internal/application/output/ports"
	"rutadeorigen/traceability/internal/domain/evidence"
	"rutadeorigen/traceability/internal/platform/exception"
)

var _ ports.EvidenceRepository = (*EvidenceRepository)(nil)

type EvidenceRepository struct {
	mu    sync.RWMutex
	items map[uuid.UUID]*evidence.Evidence
}

func NewEvidenceRepository() *EvidenceRepository {
	return &EvidenceRepository{items: make(map[uuid.UUID]*evidence.Evidence)}
}

func (r *EvidenceRepository) Save(_ context.Context, e *evidence.Evidence) error {
	if e.IsDefault() {
		return exception.New(exception.CodeInternal).WithTechnical("cannot save a default evidence")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.items[e.ID()] = e
	return nil
}

func (r *EvidenceRepository) Update(_ context.Context, e *evidence.Evidence) error {
	if e.IsDefault() {
		return exception.New(exception.CodeInternal).WithTechnical("cannot update a default evidence")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.items[e.ID()]; !ok {
		return exception.New(exception.CodeEvidenceNotFound)
	}
	r.items[e.ID()] = e
	return nil
}

func (r *EvidenceRepository) FindByID(_ context.Context, id uuid.UUID) (*evidence.Evidence, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	e, ok := r.items[id]
	if !ok {
		return evidence.Default(), nil
	}
	return e, nil
}

func (r *EvidenceRepository) ListByLotID(_ context.Context, lotID uuid.UUID) ([]*evidence.Evidence, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	items := make([]*evidence.Evidence, 0)
	for _, e := range r.items {
		if e.LotID() == lotID {
			items = append(items, e)
		}
	}

	sort.Slice(items, func(i, j int) bool {
		if !items[i].CreatedAt().Equal(items[j].CreatedAt()) {
			return items[i].CreatedAt().After(items[j].CreatedAt())
		}
		a, b := items[i].ID(), items[j].ID()
		return bytes.Compare(a[:], b[:]) > 0
	})
	return items, nil
}
