package ports

import (
	"rutadeorigen/traceability/internal/application/input/dto/request"
	"rutadeorigen/traceability/internal/application/input/dto/response"
)

type ListActorsPort interface {
	InputPort[request.ListActors, response.ActorList]
}
