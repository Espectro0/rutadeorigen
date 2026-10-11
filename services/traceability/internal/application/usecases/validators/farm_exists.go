package validators

import (
	"context"

	"github.com/google/uuid"

	"rutadeorigen/traceability/internal/application/output/ports"
	"rutadeorigen/traceability/internal/application/usecases"
	"rutadeorigen/traceability/internal/domain/farm"
	"rutadeorigen/traceability/internal/platform/exception"
)

func FarmExists(ctx context.Context, repo ports.FarmRepository, id uuid.UUID) (*farm.Farm, error) {
	f, err := repo.FindByID(ctx, id)
	if err != nil {
		return farm.Default(), exception.Wrap(err, exception.LayerApplication, "find farm by id")
	}
	if f.IsDefault() {
		return farm.Default(), usecases.ErrFarmNotFound
	}
	return f, nil
}
