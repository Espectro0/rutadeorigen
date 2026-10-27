package ports

import (
	"rutadeorigen/traceability/internal/application/input/dto/request"
	"rutadeorigen/traceability/internal/application/input/dto/response"
)

type RegisterActorPort interface {
	InputPort[request.RegisterActor, response.ActorRegistration]
}
