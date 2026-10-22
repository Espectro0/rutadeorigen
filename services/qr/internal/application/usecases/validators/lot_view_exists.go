package validators

import (
	"context"

	"github.com/google/uuid"

	"rutadeorigen/qr/internal/application/output/ports"
	"rutadeorigen/qr/internal/domain/lotview"
	"rutadeorigen/qr/internal/platform/exception"
	"rutadeorigen/qr/internal/platform/helper"
)

func LotViewExists(ctx context.Context, cache ports.LotViewCache, repo ports.LotViewRepository, id uuid.UUID) (*lotview.LotView, error) {
	if helper.IsDefaultUUID(id) {
		return lotview.Default(), exception.New(exception.CodeLotViewInvalidID)
	}

	if cached, err := cache.Get(ctx, id); err == nil && !cached.IsDefault() {
		return cached, nil
	}

	view, err := repo.FindByID(ctx, id)
	if err != nil {
		return lotview.Default(), exception.Wrap(err, "find lot view by id")
	}
	if view.IsDefault() {
		return lotview.Default(), exception.New(exception.CodeLotViewNotFound)
	}

	_ = cache.Set(ctx, view)
	return view, nil
}
