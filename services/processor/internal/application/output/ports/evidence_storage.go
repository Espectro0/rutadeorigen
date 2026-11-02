package ports

import "context"

type EvidenceStorage interface {
	Put(ctx context.Context, key, contentType string, data []byte) error
}
