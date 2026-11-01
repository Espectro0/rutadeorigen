package deletefarm

import (
	"context"

	"rutadeorigen/traceability/internal/application/input/dto/request"
	"rutadeorigen/traceability/internal/application/input/ports"
	deletefarmuc "rutadeorigen/traceability/internal/application/usecases/deletefarm"
	"rutadeorigen/traceability/internal/platform/exception"
	"rutadeorigen/traceability/internal/platform/helper"
)

var _ ports.DeleteFarmPort = (*Adapter)(nil)

type Adapter struct {
	useCase deletefarmuc.UseCase
}

func New(useCase deletefarmuc.UseCase) *Adapter {
	return &Adapter{useCase: useCase}
}

func (a *Adapter) Execute(ctx context.Context, input request.DeleteFarm) error {
	id := helper.ParseUUID(input.ID)
	if helper.IsDefaultUUID(id) {
		return exception.New(exception.CodeFarmNotFound)
	}
	return a.useCase.Execute(ctx, id)
}
