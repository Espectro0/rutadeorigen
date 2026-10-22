package ports

import (
	"context"

	"github.com/google/uuid"

	"rutadeorigen/qr/internal/domain/lotview"
)

type LotViewRepository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*lotview.LotView, error)
	Save(ctx context.Context, view *lotview.LotView) error
}
