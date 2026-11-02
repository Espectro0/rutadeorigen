package projectevidenceprocessed

import (
	"rutadeorigen/qr/internal/application/input/assembler"
	"rutadeorigen/qr/internal/application/input/dto/request"
	"rutadeorigen/qr/internal/application/input/dto/response"
	projectevidenceprocesseduc "rutadeorigen/qr/internal/application/usecases/projectevidenceprocessed"
	"rutadeorigen/qr/internal/platform/exception"
	"rutadeorigen/qr/internal/platform/helper"
)

var _ assembler.DTOAssembler[projectevidenceprocesseduc.Command, request.ProjectEvidenceProcessed, response.Empty] = Assembler{}

type Assembler struct{}

func (Assembler) ToDomain(req request.ProjectEvidenceProcessed) (projectevidenceprocesseduc.Command, error) {
	invalid := projectevidenceprocesseduc.Command{}

	messageID := helper.ParseUUID(req.MessageID)
	evidenceID := helper.ParseUUID(req.Data.EvidenceID)
	lotID := helper.ParseUUID(req.Data.LotID)
	if helper.IsDefaultUUID(messageID) || helper.IsDefaultUUID(evidenceID) || helper.IsDefaultUUID(lotID) {
		return invalid, exception.New(exception.CodeEventInvalidPayload)
	}

	processedAt := helper.DefaultDate()
	if !helper.IsEmpty(req.Data.ProcessedAt) {
		processedAt = helper.ParseDateTime(req.Data.ProcessedAt)
		if helper.IsDefaultDate(processedAt) {
			return invalid, exception.New(exception.CodeEventInvalidPayload)
		}
	}

	return projectevidenceprocesseduc.Command{
		MessageID:    messageID,
		EventType:    helper.ApplyTrim(req.EventType),
		EvidenceID:   evidenceID,
		LotID:        lotID,
		Stage:        helper.ApplyTrim(req.Data.Stage),
		FinalKey:     helper.ApplyTrim(req.Data.FinalKey),
		ThumbnailKey: helper.ApplyTrim(req.Data.ThumbnailKey),
		ContentType:  helper.ApplyTrim(req.Data.ContentType),
		ProcessedAt:  processedAt,
	}, nil
}

func (Assembler) ToDTO(_ projectevidenceprocesseduc.Command) response.Empty {
	return response.Empty{}
}
