package ports

import (
	"rutadeorigen/traceability/internal/application/input/dto/request"
	"rutadeorigen/traceability/internal/application/input/dto/response"
)

type GetLotPort interface {
	InputPort[request.GetLot, response.Lot]
}
