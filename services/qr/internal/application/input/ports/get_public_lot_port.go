package ports

import (
	"rutadeorigen/qr/internal/application/input/dto/request"
	"rutadeorigen/qr/internal/application/input/dto/response"
)

type GetPublicLotPort interface {
	InputPort[request.GetPublicLot, response.PublicLot]
}
