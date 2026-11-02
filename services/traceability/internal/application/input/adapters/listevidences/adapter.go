package listevidences

import (
	"context"

	"rutadeorigen/traceability/internal/application/input/adapters/evidencemapper"
	"rutadeorigen/traceability/internal/application/input/dto/request"
	"rutadeorigen/traceability/internal/application/input/dto/response"
	"rutadeorigen/traceability/internal/application/input/ports"
	listuc "rutadeorigen/traceability/internal/application/usecases/listevidences"
	"rutadeorigen/traceability/internal/platform/exception"
	"rutadeorigen/traceability/internal/platform/helper"
)

var _ ports.ListEvidencesPort = (*Adapter)(nil)

type Adapter struct {
	useCase listuc.UseCase
}

func New(useCase listuc.UseCase) *Adapter {
	return &Adapter{useCase: useCase}
}

func (a *Adapter) Execute(ctx context.Context, input request.ListEvidences) (response.EvidenceList, error) {
	lotID := helper.ParseUUID(input.LotID)
	if helper.IsDefaultUUID(lotID) {
		return response.EvidenceList{}, exception.New(exception.CodeLotNotFound)
	}

	items, err := a.useCase.Execute(ctx, lotID)
	if err != nil {
		return response.EvidenceList{}, err
	}

	out := make([]response.Evidence, 0, len(items))
	for _, e := range items {
		out = append(out, evidencemapper.ToResponse(e))
	}
	return response.EvidenceList{Items: out}, nil
}
