package getqrimage

import (
	"github.com/google/uuid"

	"rutadeorigen/qr/internal/application/usecases"
	"rutadeorigen/qr/internal/domain/qrcode"
)

type Query struct {
	LotID  uuid.UUID
	Size   qrcode.Size
	Format qrcode.Format
}

type UseCase interface {
	usecases.UseCaseWithReturn[Query, *qrcode.Image]
}
