package ports

import (
	"context"
	"time"
)

type PresignedUpload struct {
	URL       string
	Method    string
	Headers   map[string]string
	ExpiresAt time.Time
}

// ObjectInfo describes an object in quarantine. Exists is false when the
// object is not there; that is not an error.
type ObjectInfo struct {
	Exists    bool
	SizeBytes int64
}

type EvidenceStorage interface {
	PresignUpload(ctx context.Context, key, contentType string, ttl time.Duration) (PresignedUpload, error)
	Stat(ctx context.Context, key string) (ObjectInfo, error)
	Remove(ctx context.Context, key string) error
}
