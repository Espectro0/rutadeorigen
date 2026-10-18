package validators

import (
	"context"

	"github.com/google/uuid"

	"rutadeorigen/traceability/internal/application/output/ports"
	"rutadeorigen/traceability/internal/domain/lot"
	"rutadeorigen/traceability/internal/platform/exception"
)

func LotExists(ctx context.Context, repo ports.LotRepository, id uuid.UUID) (*lot.Lot, error) {
	l, err := repo.FindByID(ctx, id)
	if err != nil {
		return lot.Default(), exception.Wrap(err, "find lot by id")
	}
	if l.IsDefault() {
		return lot.Default(), exception.New(exception.CodeLotNotFound)
	}
	return l, nil
}
