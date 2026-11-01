package ports

import (
	"rutadeorigen/traceability/internal/application/input/dto/request"
	"rutadeorigen/traceability/internal/application/input/dto/response"
)

type VoidLotPort interface {
	InputPort[request.VoidLot, response.Lot]
}
