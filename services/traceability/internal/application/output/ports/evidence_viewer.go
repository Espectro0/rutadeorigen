package ports

import (
	"context"
	"time"
)

type ViewURL struct {
	URL       string
	ExpiresAt time.Time
}

type EvidenceViewer interface {
	PresignView(ctx context.Context, key string, ttl time.Duration) (ViewURL, error)
}
