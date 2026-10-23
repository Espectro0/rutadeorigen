package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"rutadeorigen/qr/internal/application/output/ports"
	"rutadeorigen/qr/internal/domain/qrcode"
	"rutadeorigen/qr/internal/infrastructure/output/repositories/structured/postgres/assembler"
	"rutadeorigen/qr/internal/infrastructure/output/repositories/structured/postgres/entity"
	"rutadeorigen/qr/internal/platform/exception"
)

var _ ports.QRCodeRepository = (*QRCodeRepository)(nil)

const (
	selectQRCode = `
		SELECT lot_id, content, error_correction, library_version, created_at
		FROM qr_code
		WHERE lot_id = $1`

	insertQRCode = `
		INSERT INTO qr_code (lot_id, content, error_correction, library_version, created_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (lot_id) DO NOTHING`
)

type QRCodeRepository struct {
	pool      *pgxpool.Pool
	assembler assembler.EntityAssembler[entity.QRCode, *qrcode.QRCode]
}

func NewQRCodeRepository(pool *pgxpool.Pool) *QRCodeRepository {
	return &QRCodeRepository{pool: pool, assembler: assembler.QRCode{}}
}

func (r *QRCodeRepository) FindByLotID(ctx context.Context, lotID uuid.UUID) (*qrcode.QRCode, error) {
	rows, err := conn(ctx, r.pool).Query(ctx, selectQRCode, lotID)
	if err != nil {
		return qrcode.Default(), internalError("query qr code by lot id", err)
	}

	row, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[entity.QRCode])
	if errors.Is(err, pgx.ErrNoRows) {
		return qrcode.Default(), nil
	}
	if err != nil {
		return qrcode.Default(), internalError("scan qr code", err)
	}
	return r.assembler.ToDomain(row), nil
}

func (r *QRCodeRepository) Save(ctx context.Context, code *qrcode.QRCode) error {
	if code.IsDefault() {
		return exception.New(exception.CodeInternal).WithTechnical("insert qr code: default code")
	}

	e := r.assembler.ToEntity(code)
	_, err := conn(ctx, r.pool).Exec(ctx, insertQRCode,
		e.LotID, e.Content, e.ErrorCorrection, e.LibraryVersion, e.CreatedAt,
	)
	if err != nil {
		return internalError("insert qr code", err)
	}
	return nil
}
