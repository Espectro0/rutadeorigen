package ports

import (
	"rutadeorigen/traceability/internal/application/input/dto/request"
	"rutadeorigen/traceability/internal/application/input/dto/response"
)

type CreateFarmPort interface {
	InputPort[request.CreateFarm, response.Farm]
}
