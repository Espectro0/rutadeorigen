package completeevidenceupload

import (
	"context"

	"rutadeorigen/traceability/internal/application/input/adapters/evidencemapper"
	"rutadeorigen/traceability/internal/application/input/dto/request"
	"rutadeorigen/traceability/internal/application/input/dto/response"
	"rutadeorigen/traceability/internal/application/input/ports"
	completeuc "rutadeorigen/traceability/internal/application/usecases/completeevidenceupload"
	"rutadeorigen/traceability/internal/platform/exception"
	"rutadeorigen/traceability/internal/platform/helper"
)

var _ ports.CompleteEvidenceUploadPort = (*Adapter)(nil)

type Adapter struct {
	useCase completeuc.UseCase
}

func New(useCase completeuc.UseCase) *Adapter {
	return &Adapter{useCase: useCase}
}

func (a *Adapter) Execute(ctx context.Context, input request.CompleteEvidenceUpload) (response.Evidence, error) {
	lotID := helper.ParseUUID(input.LotID)
	if helper.IsDefaultUUID(lotID) {
		return response.Evidence{}, exception.New(exception.CodeLotNotFound)
	}
	evidenceID := helper.ParseUUID(input.EvidenceID)
	if helper.IsDefaultUUID(evidenceID) {
		return response.Evidence{}, exception.New(exception.CodeEvidenceNotFound)
	}

	e, err := a.useCase.Execute(ctx, completeuc.Command{LotID: lotID, EvidenceID: evidenceID})
	if err != nil {
		return response.Evidence{}, err
	}
	return evidencemapper.ToResponse(e), nil
}
