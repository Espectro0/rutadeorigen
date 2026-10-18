package listlots

import (
	"rutadeorigen/traceability/internal/application/usecases"
	"rutadeorigen/traceability/internal/domain/lot"
)

type Query struct {
	Filter lot.Filter
	After  lot.Cursor
	Limit  int
}

type UseCase interface {
	usecases.UseCaseWithReturn[Query, lot.Page]
}
