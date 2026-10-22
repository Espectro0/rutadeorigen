package ports

import (
	"rutadeorigen/traceability/internal/application/input/dto/request"
	"rutadeorigen/traceability/internal/application/input/dto/response"
)

type ListFarmsPort interface {
	InputPort[request.ListFarms, response.FarmList]
}
