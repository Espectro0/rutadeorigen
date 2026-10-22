package ports

import (
	"context"

	"github.com/google/uuid"

	"rutadeorigen/qr/internal/domain/qrcode"
)

type QRCodeRepository interface {
	FindByLotID(ctx context.Context, lotID uuid.UUID) (*qrcode.QRCode, error)
	Save(ctx context.Context, code *qrcode.QRCode) error
}
