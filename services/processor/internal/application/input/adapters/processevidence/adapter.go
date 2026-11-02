package processevidence

import (
	"context"

	"rutadeorigen/processor/internal/application/input/dto/request"
	"rutadeorigen/processor/internal/application/input/ports"
	processuc "rutadeorigen/processor/internal/application/usecases/processevidence"
	"rutadeorigen/processor/internal/platform/exception"
	"rutadeorigen/processor/internal/platform/helper"
)

var _ ports.ProcessEvidencePort = (*Adapter)(nil)

type Adapter struct {
	useCase processuc.UseCase
}

func New(useCase processuc.UseCase) *Adapter {
	return &Adapter{useCase: useCase}
}

func (a *Adapter) Execute(ctx context.Context, input request.ProcessEvidence) error {
	messageID := helper.ParseUUID(input.MessageID)
	evidenceID := helper.ParseUUID(input.EvidenceID)
	lotID := helper.ParseUUID(input.LotID)
	uploaderID := helper.ParseUUID(input.UploaderActorID)

	if helper.IsDefaultUUID(messageID) || helper.IsDefaultUUID(evidenceID) ||
		helper.IsDefaultUUID(lotID) || helper.IsDefaultUUID(uploaderID) {
		return exception.New(exception.CodeEventInvalidPayload)
	}

	return a.useCase.Execute(ctx, processuc.Command{
		MessageID:     messageID,
		EventType:     helper.ApplyTrim(input.EventType),
		EvidenceID:    evidenceID,
		LotID:         lotID,
		LotCode:       helper.ApplyTrim(input.LotCode),
		Stage:         helper.ApplyTrim(input.Stage),
		FileName:      helper.ApplyTrim(input.FileName),
		QuarantineKey: helper.ApplyTrim(input.QuarantineKey),
		SizeBytes:     input.SizeBytes,
		Uploader: processuc.Uploader{
			ActorID: uploaderID,
			Role:    helper.ApplyTrim(input.UploaderRole),
			Name:    helper.ApplyTrim(input.UploaderName),
			Email:   helper.ApplyTrim(input.UploaderEmail),
		},
	})
}
