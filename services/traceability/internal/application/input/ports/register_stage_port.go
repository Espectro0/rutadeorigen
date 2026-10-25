package ports

import (
	"rutadeorigen/traceability/internal/application/input/dto/request"
	"rutadeorigen/traceability/internal/application/input/dto/response"
)

type RegisterStagePort interface {
	InputPort[request.RegisterStage, response.RegisteredStage]
}
