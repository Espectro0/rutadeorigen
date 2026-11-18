package ports

import (
	"context"

	"github.com/google/uuid"

	"rutadeorigen/qr/internal/domain/stageview"
)

type StageViewRepository interface {
	Save(ctx context.Context, view *stageview.StageView) error
	ListByLotID(ctx context.Context, lotID uuid.UUID) ([]*stageview.StageView, error)
}
