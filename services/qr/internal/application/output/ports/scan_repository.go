package ports

import (
	"context"
	"time"

	"github.com/google/uuid"

	"rutadeorigen/qr/internal/domain/scan"
)

type ScanRepository interface {
	Save(ctx context.Context, scan *scan.Scan) error
	StatsByLot(ctx context.Context, lotID uuid.UUID, since time.Time) (*scan.Stats, error)
}
