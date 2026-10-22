package ports

import (
	"context"

	"github.com/google/uuid"

	"rutadeorigen/qr/internal/domain/lotview"
)

type LotViewCache interface {
	Get(ctx context.Context, id uuid.UUID) (*lotview.LotView, error)
	Set(ctx context.Context, view *lotview.LotView) error
	Delete(ctx context.Context, id uuid.UUID) error
}
