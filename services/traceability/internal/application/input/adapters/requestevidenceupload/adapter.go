package requestevidenceupload

import (
	"context"
	"time"

	"rutadeorigen/traceability/internal/application/input/adapters/evidencemapper"
	"rutadeorigen/traceability/internal/application/input/dto/request"
	"rutadeorigen/traceability/internal/application/input/dto/response"
	"rutadeorigen/traceability/internal/application/input/ports"
	requestuc "rutadeorigen/traceability/internal/application/usecases/requestevidenceupload"
	"rutadeorigen/traceability/internal/platform/exception"
	"rutadeorigen/traceability/internal/platform/helper"
)

var _ ports.RequestEvidenceUploadPort = (*Adapter)(nil)

type Adapter struct {
	useCase requestuc.UseCase
}

func New(useCase requestuc.UseCase) *Adapter {
	return &Adapter{useCase: useCase}
}

func (a *Adapter) Execute(ctx context.Context, input request.EvidenceUpload) (response.EvidenceUpload, error) {
	lotID := helper.ParseUUID(input.LotID)
	if helper.IsDefaultUUID(lotID) {
		return response.EvidenceUpload{}, exception.New(exception.CodeLotNotFound)
	}

	res, err := a.useCase.Execute(ctx, requestuc.Command{
		LotID:       lotID,
		FileName:    input.FileName,
		ContentType: input.ContentType,
		SizeBytes:   input.SizeBytes,
	})
	if err != nil {
		return response.EvidenceUpload{}, err
	}

	return response.EvidenceUpload{
		Evidence: evidencemapper.ToResponse(res.Evidence),
		Upload: response.EvidenceUploadTarget{
			URL:       res.Upload.URL,
			Method:    res.Upload.Method,
			Headers:   res.Upload.Headers,
			ExpiresAt: res.Upload.ExpiresAt.Format(time.RFC3339),
		},
	}, nil
}
