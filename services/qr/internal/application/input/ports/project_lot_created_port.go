package ports

import (
	"rutadeorigen/qr/internal/application/input/dto/request"
	"rutadeorigen/qr/internal/application/input/dto/response"
)

type ProjectLotCreatedPort interface {
	InputPort[request.ProjectLotCreated, response.Empty]
}
