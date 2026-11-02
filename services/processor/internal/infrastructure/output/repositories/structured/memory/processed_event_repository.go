package memory

import (
	"context"
	"sync"

	"github.com/google/uuid"

	"rutadeorigen/processor/internal/application/output/ports"
)

var _ ports.ProcessedEventRepository = (*ProcessedEventRepository)(nil)

type ProcessedEventRepository struct {
	mu   sync.Mutex
	seen map[uuid.UUID]string
}

func NewProcessedEventRepository() *ProcessedEventRepository {
	return &ProcessedEventRepository{seen: make(map[uuid.UUID]string)}
}

func (r *ProcessedEventRepository) Exists(_ context.Context, messageID uuid.UUID) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, found := r.seen[messageID]
	return found, nil
}

func (r *ProcessedEventRepository) Register(_ context.Context, messageID uuid.UUID, eventType string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, found := r.seen[messageID]; found {
		return false, nil
	}
	r.seen[messageID] = eventType
	return true, nil
}
