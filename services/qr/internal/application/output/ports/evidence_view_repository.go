package ports

import (
	"context"

	"github.com/google/uuid"

	"rutadeorigen/qr/internal/domain/evidenceview"
)

type EvidenceViewRepository interface {
	Save(ctx context.Context, view *evidenceview.EvidenceView) error
	ListByLotID(ctx context.Context, lotID uuid.UUID, limit int) ([]*evidenceview.EvidenceView, error)
}
