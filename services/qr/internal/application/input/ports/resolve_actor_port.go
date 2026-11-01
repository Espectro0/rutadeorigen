package ports

import (
	"rutadeorigen/qr/internal/application/input/dto/request"
	"rutadeorigen/qr/internal/application/input/dto/response"
)

type ResolveActorPort interface {
	InputPort[request.ResolveActor, response.ResolvedActor]
}
