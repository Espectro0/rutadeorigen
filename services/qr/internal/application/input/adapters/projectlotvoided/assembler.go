package projectlotvoided

import (
	"rutadeorigen/qr/internal/application/input/assembler"
	"rutadeorigen/qr/internal/application/input/dto/request"
	"rutadeorigen/qr/internal/application/input/dto/response"
	projectlotvoideduc "rutadeorigen/qr/internal/application/usecases/projectlotvoided"
	"rutadeorigen/qr/internal/platform/exception"
	"rutadeorigen/qr/internal/platform/helper"
)

var _ assembler.DTOAssembler[projectlotvoideduc.Command, request.ProjectLotVoided, response.Empty] = Assembler{}

type Assembler struct{}

func (Assembler) ToDomain(req request.ProjectLotVoided) (projectlotvoideduc.Command, error) {
	messageID := helper.ParseUUID(req.MessageID)
	if helper.IsDefaultUUID(messageID) {
		return projectlotvoideduc.Command{}, exception.New(exception.CodeEventInvalidPayload)
	}

	lotID := helper.ParseUUID(req.Data.LotID)
	if helper.IsDefaultUUID(lotID) {
		return projectlotvoideduc.Command{}, exception.New(exception.CodeEventInvalidPayload)
	}

	occurredAt := helper.DefaultDate()
	if !helper.IsEmpty(req.OccurredAt) {
		occurredAt = helper.ParseDateTime(req.OccurredAt)
		if helper.IsDefaultDate(occurredAt) {
			return projectlotvoideduc.Command{}, exception.New(exception.CodeEventInvalidPayload)
		}
	}

	return projectlotvoideduc.Command{
		MessageID:  messageID,
		EventType:  helper.ApplyTrim(req.EventType),
		LotID:      lotID,
		Reason:     helper.ApplyTrim(req.Data.Reason),
		Version:    req.Version,
		OccurredAt: occurredAt,
	}, nil
}

func (Assembler) ToDTO(_ projectlotvoideduc.Command) response.Empty {
	return response.Empty{}
}
