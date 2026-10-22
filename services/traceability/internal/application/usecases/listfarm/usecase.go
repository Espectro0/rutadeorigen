package listfarm

import (
	"rutadeorigen/traceability/internal/application/usecases"
	"rutadeorigen/traceability/internal/domain/farm"
)

type Query struct{}

type UseCase interface {
	usecases.UseCaseWithReturn[Query, []*farm.Farm]
}
