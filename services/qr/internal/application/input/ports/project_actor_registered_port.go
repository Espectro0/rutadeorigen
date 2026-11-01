package ports

import (
	"rutadeorigen/qr/internal/application/input/dto/request"
	"rutadeorigen/qr/internal/application/input/dto/response"
)

type ProjectActorRegisteredPort interface {
	InputPort[request.ProjectActorRegistered, response.Empty]
}
