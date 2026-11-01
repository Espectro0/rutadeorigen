package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"rutadeorigen/qr/internal/application/output/ports"
	"rutadeorigen/qr/internal/domain/scan"
	"rutadeorigen/qr/internal/platform/exception"
	"rutadeorigen/qr/internal/platform/helper"
)

var _ ports.ScanRepository = (*ScanRepository)(nil)

const (
	insertScan = `INSERT INTO qr_scan (lot_id, scanned_at) VALUES ($1, $2)`

	selectScanTotals = `
		SELECT count(*), min(scanned_at), max(scanned_at)
		FROM qr_scan
		WHERE lot_id = $1`

	selectScanDaily = `
		SELECT (scanned_at AT TIME ZONE 'UTC')::date AS day, count(*)
		FROM qr_scan
		WHERE lot_id = $1 AND scanned_at >= $2
		GROUP BY day
		ORDER BY day`
)

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

func (r *ScanRepository) StatsByLot(ctx context.Context, lotID uuid.UUID, since time.Time) (*scan.Stats, error) {
	var (
		total       int
		first, last *time.Time
	)
	if err := conn(ctx, r.pool).QueryRow(ctx, selectScanTotals, lotID).Scan(&total, &first, &last); err != nil {
		return scan.NewStats(lotID, 0, helper.DefaultDate(), helper.DefaultDate(), nil), internalError("query scan totals", err)
	}

	rows, err := conn(ctx, r.pool).Query(ctx, selectScanDaily, lotID, since.UTC())
	if err != nil {
		return scan.NewStats(lotID, 0, helper.DefaultDate(), helper.DefaultDate(), nil), internalError("query daily scans", err)
	}
	defer rows.Close()

	daily := make([]scan.DailyCount, 0)
	for rows.Next() {
		var (
			day   time.Time
			count int
		)
		if err := rows.Scan(&day, &count); err != nil {
			return scan.NewStats(lotID, 0, helper.DefaultDate(), helper.DefaultDate(), nil), internalError("scan daily count", err)
		}
		daily = append(daily, scan.DailyCount{Day: helper.DateUTC(day), Count: count})
	}
	if err := rows.Err(); err != nil {
		return scan.NewStats(lotID, 0, helper.DefaultDate(), helper.DefaultDate(), nil), internalError("read daily scans", err)
	}

	return scan.NewStats(lotID, total, helper.DateOrDefault(first), helper.DateOrDefault(last), daily), nil
}
