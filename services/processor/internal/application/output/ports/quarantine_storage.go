package ports

import "context"

type QuarantineObject struct {
	Exists bool
	Data   []byte
}

type QuarantineStorage interface {
	Get(ctx context.Context, key string, maxBytes int64) (QuarantineObject, error)
	Remove(ctx context.Context, key string) error
}
