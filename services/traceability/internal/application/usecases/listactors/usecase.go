package listactors

import (
	"rutadeorigen/traceability/internal/application/usecases"
	"rutadeorigen/traceability/internal/domain/actor"
)

type Query struct {
	Type actor.Type
}

type UseCase interface {
	usecases.UseCaseWithReturn[Query, []*actor.Actor]
}
