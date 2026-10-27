package ports

import (
	"rutadeorigen/traceability/internal/application/input/dto/request"
	"rutadeorigen/traceability/internal/application/input/dto/response"
)

type ResolveActorPort interface {
	InputPort[request.ResolveActor, response.ResolvedActor]
}
