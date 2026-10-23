package assembler

import (
	"rutadeorigen/qr/internal/domain/qrcode"
	"rutadeorigen/qr/internal/infrastructure/output/repositories/structured/postgres/entity"
)

var _ EntityAssembler[entity.QRCode, *qrcode.QRCode] = QRCode{}

type QRCode struct{}

func (QRCode) ToDomain(e entity.QRCode) *qrcode.QRCode {
	return qrcode.Restore(qrcode.RestoreParams{
		LotID:           e.LotID,
		Content:         e.Content,
		ErrorCorrection: qrcode.ParseErrorCorrection(e.ErrorCorrection),
		LibraryVersion:  e.LibraryVersion,
		CreatedAt:       e.CreatedAt.UTC(),
	})
}

func (QRCode) ToEntity(c *qrcode.QRCode) entity.QRCode {
	if c.IsDefault() {
		return entity.QRCode{}
	}

	return entity.QRCode{
		LotID:           c.LotID(),
		Content:         c.Content(),
		ErrorCorrection: c.ErrorCorrection().String(),
		LibraryVersion:  c.LibraryVersion(),
		CreatedAt:       c.CreatedAt().UTC(),
	}
}
