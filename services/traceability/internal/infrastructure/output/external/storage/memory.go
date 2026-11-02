package storage

import (
	"context"
	"sync"
	"time"

	"rutadeorigen/traceability/internal/application/output/ports"
)

var (
	_ ports.EvidenceStorage = (*Memory)(nil)
	_ ports.EvidenceViewer  = (*Memory)(nil)
)

type Memory struct {
	mu      sync.RWMutex
	objects map[string]int64
}

func NewMemory() *Memory {
	return &Memory{objects: make(map[string]int64)}
}

func (m *Memory) Put(key string, sizeBytes int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.objects[key] = sizeBytes
}

func (m *Memory) PresignUpload(_ context.Context, key, contentType string, ttl time.Duration) (ports.PresignedUpload, error) {
	return ports.PresignedUpload{
		URL:       "http://memory.invalid/quarantine/" + key,
		Method:    "PUT",
		Headers:   map[string]string{"Content-Type": contentType},
		ExpiresAt: time.Now().UTC().Add(ttl),
	}, nil
}

func (m *Memory) PresignView(_ context.Context, key string, ttl time.Duration) (ports.ViewURL, error) {
	return ports.ViewURL{
		URL:       "http://memory.invalid/evidences/" + key,
		ExpiresAt: time.Now().UTC().Add(ttl),
	}, nil
}

func (m *Memory) Stat(_ context.Context, key string) (ports.ObjectInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	size, ok := m.objects[key]
	if !ok {
		return ports.ObjectInfo{}, nil
	}
	return ports.ObjectInfo{Exists: true, SizeBytes: size}, nil
}

func (m *Memory) Remove(_ context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.objects, key)
	return nil
}
