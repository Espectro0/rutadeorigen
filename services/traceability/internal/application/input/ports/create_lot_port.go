package ports

import (
	"rutadeorigen/traceability/internal/application/input/dto/request"
	"rutadeorigen/traceability/internal/application/input/dto/response"
)

type CreateLotPort interface {
	InputPort[request.CreateLot, response.Lot]
}
