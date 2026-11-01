package ports

import (
	"rutadeorigen/qr/internal/application/input/dto/request"
	"rutadeorigen/qr/internal/application/input/dto/response"
)

type ProjectLotVoidedPort interface {
	InputPort[request.ProjectLotVoided, response.Empty]
}
