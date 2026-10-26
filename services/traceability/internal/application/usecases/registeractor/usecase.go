package registeractor

import (
	"rutadeorigen/traceability/internal/application/usecases"
	"rutadeorigen/traceability/internal/domain/actor"
)

type Command struct {
	UserID string
	Email  string
	Type   actor.Type
	Name   string
}

type Result struct {
	Actor   *actor.Actor
	Created bool
}

type UseCase interface {
	usecases.UseCaseWithReturn[Command, Result]
}
