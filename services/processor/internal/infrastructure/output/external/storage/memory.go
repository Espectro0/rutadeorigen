package storage

import (
	"context"
	"sync"

	"rutadeorigen/processor/internal/application/output/ports"
)

var (
	_ ports.QuarantineStorage = (*MemoryQuarantine)(nil)
	_ ports.EvidenceStorage   = (*MemoryEvidences)(nil)
)

type MemoryQuarantine struct {
	mu      sync.RWMutex
	objects map[string][]byte
}

func NewMemoryQuarantine() *MemoryQuarantine {
	return &MemoryQuarantine{objects: make(map[string][]byte)}
}

func (m *MemoryQuarantine) Seed(key string, data []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.objects[key] = data
}

func (m *MemoryQuarantine) Get(_ context.Context, key string, maxBytes int64) (ports.QuarantineObject, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	data, found := m.objects[key]
	if !found {
		return ports.QuarantineObject{}, nil
	}
	if int64(len(data)) > maxBytes {
		data = data[:maxBytes]
	}
	return ports.QuarantineObject{Exists: true, Data: data}, nil
}

func (m *MemoryQuarantine) Remove(_ context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.objects, key)
	return nil
}

type MemoryEvidences struct {
	mu      sync.RWMutex
	objects map[string][]byte
}

func NewMemoryEvidences() *MemoryEvidences {
	return &MemoryEvidences{objects: make(map[string][]byte)}
}

func (m *MemoryEvidences) Put(_ context.Context, key, _ string, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.objects[key] = data
	return nil
}

func (m *MemoryEvidences) Keys() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	keys := make([]string, 0, len(m.objects))
	for key := range m.objects {
		keys = append(keys, key)
	}
	return keys
}
