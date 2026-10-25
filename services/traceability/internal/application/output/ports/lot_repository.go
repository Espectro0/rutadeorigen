package ports

import (
	"context"

	"github.com/google/uuid"

	"rutadeorigen/traceability/internal/domain/lot"
)

type LotRepository interface {
	ExistsByCode(ctx context.Context, farmID uuid.UUID, code string) (bool, error)
	UpdateStage(ctx context.Context, lot *lot.Lot, expectedVersion int) error
	Save(ctx context.Context, lot *lot.Lot) error
	FindByID(ctx context.Context, id uuid.UUID) (*lot.Lot, error)
	List(ctx context.Context, filter lot.Filter, after lot.Cursor, limit int) ([]*lot.Lot, error)
}
