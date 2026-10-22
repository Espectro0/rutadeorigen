package ports

import (
	"context"

	"rutadeorigen/qr/internal/domain/scan"
)

type ScanRepository interface {
	Save(ctx context.Context, scan *scan.Scan) error
}
