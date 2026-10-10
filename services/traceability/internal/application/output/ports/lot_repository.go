package ports

import (
	"context"

	"github.com/google/uuid"

	"rutadeorigen/traceability/internal/domain/lot"
)

type LotRepository interface {
	ExistsByCode(ctx context.Context, farmId uuid.UUID, code string) (bool, error)
	Save(ctx context.Context, lot *lot.Lot) error
}
