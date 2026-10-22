package ports

import (
	"context"

	"rutadeorigen/qr/internal/domain/qrcode"
)

type RenderRequest struct {
	Content         string
	ErrorCorrection qrcode.ErrorCorrection
	Size            qrcode.Size
	Format          qrcode.Format
}

type QRRenderer interface {
	Render(ctx context.Context, request RenderRequest) ([]byte, error)
	LibraryVersion() string
}
