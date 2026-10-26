package projectlotstage

import (
	"rutadeorigen/qr/internal/application/input/assembler"
	"rutadeorigen/qr/internal/application/input/dto/request"
	"rutadeorigen/qr/internal/application/input/dto/response"
	projectlotstageuc "rutadeorigen/qr/internal/application/usecases/projectlotstage"
	"rutadeorigen/qr/internal/domain/lotview"
	"rutadeorigen/qr/internal/platform/exception"
	"rutadeorigen/qr/internal/platform/helper"
)

var _ assembler.DTOAssembler[projectlotstageuc.Command, request.ProjectLotStage, response.Empty] = Assembler{}

type Assembler struct{}

func (Assembler) ToDomain(req request.ProjectLotStage) (projectlotstageuc.Command, error) {
	messageID := helper.ParseUUID(req.MessageID)
	if helper.IsDefaultUUID(messageID) {
		return projectlotstageuc.Command{}, exception.New(exception.CodeEventInvalidPayload)
	}

	lotID := helper.ParseUUID(req.Data.LotID)
	if helper.IsDefaultUUID(lotID) {
		return projectlotstageuc.Command{}, exception.New(exception.CodeEventInvalidPayload)
	}

	occurredAt := helper.DefaultDate()
	if !helper.IsEmpty(req.OccurredAt) {
		occurredAt = helper.ParseDateTime(req.OccurredAt)
		if helper.IsDefaultDate(occurredAt) {
			return projectlotstageuc.Command{}, exception.New(exception.CodeEventInvalidPayload)
		}
	}

	return projectlotstageuc.Command{
		MessageID:  messageID,
		EventType:  helper.ApplyTrim(req.EventType),
		LotID:      lotID,
		Stage:      lotview.ParseStage(req.Data.Stage),
		Version:    req.Version,
		OccurredAt: occurredAt,
	}, nil
}

func (Assembler) ToDTO(_ projectlotstageuc.Command) response.Empty {
	return response.Empty{}
}
