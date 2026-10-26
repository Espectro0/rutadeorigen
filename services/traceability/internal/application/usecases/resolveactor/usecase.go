package resolveactor

import (
	"rutadeorigen/traceability/internal/application/usecases"
	"rutadeorigen/traceability/internal/domain/actor"
)

type UseCase interface {
	usecases.UseCaseWithReturn[string, *actor.Actor]
}
