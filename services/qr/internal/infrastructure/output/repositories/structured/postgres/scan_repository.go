package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"rutadeorigen/qr/internal/application/output/ports"
	"rutadeorigen/qr/internal/domain/scan"
	"rutadeorigen/qr/internal/platform/exception"
)

var _ ports.ScanRepository = (*ScanRepository)(nil)

const insertScan = `INSERT INTO qr_scan (lot_id, scanned_at) VALUES ($1, $2)`

type ScanRepository struct{ pool *pgxpool.Pool }

func NewScanRepository(pool *pgxpool.Pool) *ScanRepository {
	return &ScanRepository{pool: pool}
}

func (r *ScanRepository) Save(ctx context.Context, s *scan.Scan) error {
	if s.IsDefault() {
		return exception.New(exception.CodeInternal).WithTechnical("insert scan: default scan")
	}

	if _, err := conn(ctx, r.pool).Exec(ctx, insertScan, s.LotID(), s.ScannedAt().UTC()); err != nil {
		return internalError("insert scan", err)
	}
	return nil
}
