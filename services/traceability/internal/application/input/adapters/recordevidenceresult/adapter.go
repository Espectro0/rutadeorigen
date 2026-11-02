package recordevidenceresult

import (
	"context"
	"time"

	"rutadeorigen/traceability/internal/application/input/dto/request"
	"rutadeorigen/traceability/internal/application/input/ports"
	recorduc "rutadeorigen/traceability/internal/application/usecases/recordevidenceresult"
	"rutadeorigen/traceability/internal/platform/exception"
	"rutadeorigen/traceability/internal/platform/helper"
)

var _ ports.RecordEvidenceResultPort = (*Adapter)(nil)

type Adapter struct {
	useCase recorduc.UseCase
}

func New(useCase recorduc.UseCase) *Adapter {
	return &Adapter{useCase: useCase}
}

func (a *Adapter) Execute(ctx context.Context, input request.RecordEvidenceResult) error {
	evidenceID := helper.ParseUUID(input.EvidenceID)
	if helper.IsDefaultUUID(evidenceID) {
		return exception.New(exception.CodeEventInvalidPayload)
	}

	at, err := time.Parse(time.RFC3339, helper.ApplyTrim(input.ProcessedAt))
	if err != nil {
		return exception.New(exception.CodeEventInvalidPayload)
	}

	cmd := recorduc.Command{
		EvidenceID:   evidenceID,
		SHA256:       input.SHA256,
		FinalKey:     input.FinalKey,
		ThumbnailKey: input.ThumbnailKey,
		Reason:       input.Reason,
		At:           at,
	}
	switch input.Type {
	case "evidence.processed":
		cmd.Outcome = recorduc.OutcomeProcessed
	case "evidence.rejected":
		cmd.Outcome = recorduc.OutcomeRejected
	default:
		return exception.New(exception.CodeEventInvalidPayload)
	}

	return a.useCase.Execute(ctx, cmd)
}
