package ports

import (
	"context"

	"github.com/google/uuid"

	"rutadeorigen/traceability/internal/domain/lot"
)

type LotEventRepository interface {
	Save(ctx context.Context, event *lot.Event) error
	FindLast(ctx context.Context, lotID uuid.UUID) (*lot.Event, error)
	ListByLotID(ctx context.Context, lotID uuid.UUID) ([]*lot.Event, error)
}
