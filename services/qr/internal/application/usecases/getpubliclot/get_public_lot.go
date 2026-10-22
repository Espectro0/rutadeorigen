package getpubliclot

import (
	"context"

	"github.com/google/uuid"

	"rutadeorigen/qr/internal/application/output/ports"
	"rutadeorigen/qr/internal/application/usecases/validators"
	"rutadeorigen/qr/internal/domain/lotview"
)

var _ UseCase = (*GetPublicLot)(nil)

type GetPublicLot struct {
	views ports.LotViewRepository
	cache ports.LotViewCache
}

func New(views ports.LotViewRepository, cache ports.LotViewCache) *GetPublicLot {
	return &GetPublicLot{views: views, cache: cache}
}

func (uc *GetPublicLot) Execute(ctx context.Context, id uuid.UUID) (*lotview.LotView, error) {
	return validators.LotViewExists(ctx, uc.cache, uc.views, id)
}
