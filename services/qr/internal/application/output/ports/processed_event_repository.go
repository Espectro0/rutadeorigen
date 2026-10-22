package ports

import (
	"context"

	"github.com/google/uuid"
)

type ProcessedEventRepository interface {
	Register(ctx context.Context, messageID uuid.UUID, eventType string) (bool, error)
}
