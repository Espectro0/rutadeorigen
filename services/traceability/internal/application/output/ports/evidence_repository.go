package ports

import (
	"context"

	"github.com/google/uuid"

	"rutadeorigen/traceability/internal/domain/evidence"
)

type EvidenceRepository interface {
	Save(ctx context.Context, e *evidence.Evidence) error
	// Update persists the status and the processing result of an existing evidence.
	Update(ctx context.Context, e *evidence.Evidence) error
	FindByID(ctx context.Context, id uuid.UUID) (*evidence.Evidence, error)
	ListByLotID(ctx context.Context, lotID uuid.UUID) ([]*evidence.Evidence, error)
}
