package memory

import (
	"context"
	"sort"
	"sync"

	"github.com/google/uuid"

	"rutadeorigen/traceability/internal/application/output/ports"
	"rutadeorigen/traceability/internal/domain/lot"
	"rutadeorigen/traceability/internal/platform/exception"
)

var _ ports.LotEventRepository = (*LotEventRepository)(nil)

type LotEventRepository struct {
	mu     sync.RWMutex
	byLot  map[uuid.UUID][]*lot.Event
	ids    map[uuid.UUID]struct{}
	hashes map[string]struct{}
}

func NewLotEventRepository(seed ...*lot.Event) *LotEventRepository {
	r := &LotEventRepository{
		byLot:  make(map[uuid.UUID][]*lot.Event),
		ids:    make(map[uuid.UUID]struct{}),
		hashes: make(map[string]struct{}),
	}
	for _, e := range seed {
		_ = r.insert(e)
	}
	return r
}

func (r *LotEventRepository) Save(_ context.Context, e *lot.Event) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.insert(e)
}

func (r *LotEventRepository) FindLast(_ context.Context, lotID uuid.UUID) (*lot.Event, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	events := r.byLot[lotID]
	if len(events) == 0 {
		return lot.DefaultEvent(), nil
	}
	return cloneEvent(events[len(events)-1]), nil
}

func (r *LotEventRepository) ListByLotID(_ context.Context, lotID uuid.UUID) ([]*lot.Event, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	events := r.byLot[lotID]
	result := make([]*lot.Event, len(events))
	for i, e := range events {
		result[i] = cloneEvent(e)
	}
	return result, nil
}

func (r *LotEventRepository) insert(e *lot.Event) error {
	if e.IsDefault() {
		return exception.New(exception.CodeInternal).WithTechnical("cannot save a default lot event")
	}
	if e.Sequence() < 1 {
		return exception.New(exception.CodeInternal).WithTechnical("lot event sequence must be positive")
	}

	if _, exists := r.ids[e.ID()]; exists {
		return exception.New(exception.CodeInternal).WithTechnical("duplicated lot event id")
	}
	if _, exists := r.hashes[e.Hash()]; exists {
		return exception.New(exception.CodeInternal).WithTechnical("duplicated lot event hash")
	}
	for _, stored := range r.byLot[e.LotID()] {
		if stored.Sequence() == e.Sequence() {
			return exception.New(exception.CodeLotVersionConflict)
		}
	}

	events := append(r.byLot[e.LotID()], cloneEvent(e))
	sort.Slice(events, func(i, j int) bool { return events[i].Sequence() < events[j].Sequence() })

	r.byLot[e.LotID()] = events
	r.ids[e.ID()] = struct{}{}
	r.hashes[e.Hash()] = struct{}{}
	return nil
}

func cloneEvent(e *lot.Event) *lot.Event {
	return lot.RestoreEvent(lot.RestoreEventParams{
		ID:           e.ID(),
		LotID:        e.LotID(),
		Sequence:     e.Sequence(),
		EventType:    e.Type(),
		Stage:        e.Stage(),
		EventDate:    e.EventDate(),
		Note:         e.Note(),
		ActorID:      e.ActorID(),
		ActorRole:    e.ActorRole(),
		Details:      e.Details(),
		PreviousHash: e.PreviousHash(),
		Hash:         e.Hash(),
		RegisteredAt: e.RegisteredAt(),
	})
}
