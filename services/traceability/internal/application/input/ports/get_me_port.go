package ports

import (
	"rutadeorigen/traceability/internal/application/input/dto/request"
	"rutadeorigen/traceability/internal/application/input/dto/response"
)

type GetMePort interface {
	InputPort[request.GetMe, response.Me]
}
