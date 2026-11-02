package storage

import (
	"context"
	"time"

	"rutadeorigen/qr/internal/application/output/ports"
)

var _ ports.EvidenceViewer = (*Memory)(nil)

type Memory struct{}

func NewMemory() *Memory {
	return &Memory{}
}

func (*Memory) PresignView(_ context.Context, key string, ttl time.Duration) (ports.ViewURL, error) {
	return ports.ViewURL{
		URL:       "http://memory.invalid/evidences/" + key,
		ExpiresAt: time.Now().UTC().Add(ttl),
	}, nil
}
