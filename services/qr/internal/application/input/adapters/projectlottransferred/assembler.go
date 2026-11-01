package projectlottransferred

import (
	"rutadeorigen/qr/internal/application/input/assembler"
	"rutadeorigen/qr/internal/application/input/dto/request"
	"rutadeorigen/qr/internal/application/input/dto/response"
	projectlottransferreduc "rutadeorigen/qr/internal/application/usecases/projectlottransferred"
	"rutadeorigen/qr/internal/platform/exception"
	"rutadeorigen/qr/internal/platform/helper"
)

var _ assembler.DTOAssembler[projectlottransferreduc.Command, request.ProjectLotTransferred, response.Empty] = Assembler{}

type Assembler struct{}

func (Assembler) ToDomain(req request.ProjectLotTransferred) (projectlottransferreduc.Command, error) {
	messageID := helper.ParseUUID(req.MessageID)
	if helper.IsDefaultUUID(messageID) {
		return projectlottransferreduc.Command{}, exception.New(exception.CodeEventInvalidPayload)
	}

	lotID := helper.ParseUUID(req.Data.LotID)
	if helper.IsDefaultUUID(lotID) {
		return projectlottransferreduc.Command{}, exception.New(exception.CodeEventInvalidPayload)
	}

	custodianID := helper.ParseUUID(req.Data.CustodianID)
	if helper.IsDefaultUUID(custodianID) {
		return projectlottransferreduc.Command{}, exception.New(exception.CodeEventInvalidPayload)
	}

	occurredAt := helper.DefaultDate()
	if !helper.IsEmpty(req.OccurredAt) {
		occurredAt = helper.ParseDateTime(req.OccurredAt)
		if helper.IsDefaultDate(occurredAt) {
			return projectlottransferreduc.Command{}, exception.New(exception.CodeEventInvalidPayload)
		}
	}

	return projectlottransferreduc.Command{
		MessageID:   messageID,
		EventType:   helper.ApplyTrim(req.EventType),
		LotID:       lotID,
		CustodianID: custodianID,
		Version:     req.Version,
		OccurredAt:  occurredAt,
	}, nil
}

func (Assembler) ToDTO(_ projectlottransferreduc.Command) response.Empty {
	return response.Empty{}
}
