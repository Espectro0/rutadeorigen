package ports

import (
	"rutadeorigen/qr/internal/application/input/dto/request"
	"rutadeorigen/qr/internal/application/input/dto/response"
)

type GetQRImagePort interface {
	InputPort[request.GetQRImage, response.QRImage]
}
