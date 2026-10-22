package ports

import (
	"context"

	"github.com/google/uuid"

	"rutadeorigen/traceability/internal/domain/farm"
)

type FarmRepository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*farm.Farm, error)
	Save(ctx context.Context, farm *farm.Farm) error
	ListByProducer(ctx context.Context, producerID uuid.UUID) ([]*farm.Farm, error)
}
