package ports

import (
	"context"

	"github.com/google/uuid"
)

type ProcessedEventRepository interface {
	Exists(ctx context.Context, messageID uuid.UUID) (bool, error)
	Register(ctx context.Context, messageID uuid.UUID, eventType string) (bool, error)
}
