package projectlotcreated

import (
	"time"

	"rutadeorigen/qr/internal/application/input/assembler"
	"rutadeorigen/qr/internal/application/input/dto/request"
	"rutadeorigen/qr/internal/application/input/dto/response"
	projectlotcreateduc "rutadeorigen/qr/internal/application/usecases/projectlotcreated"
	"rutadeorigen/qr/internal/domain/lotview"
	"rutadeorigen/qr/internal/platform/exception"
	"rutadeorigen/qr/internal/platform/helper"
)

var _ assembler.DTOAssembler[projectlotcreateduc.Command, request.ProjectLotCreated, response.Empty] = Assembler{}

type Assembler struct{}

func (Assembler) ToDomain(req request.ProjectLotCreated) (projectlotcreateduc.Command, error) {
	messageID := helper.ParseUUID(req.MessageID)
	if helper.IsDefaultUUID(messageID) {
		return projectlotcreateduc.Command{}, exception.New(exception.CodeEventInvalidPayload)
	}

	occurredAt := helper.DefaultDate()
	if !helper.IsEmpty(req.OccurredAt) {
		occurredAt = helper.ParseDateTime(req.OccurredAt)
		if helper.IsDefaultDate(occurredAt) {
			return projectlotcreateduc.Command{}, exception.New(exception.CodeEventInvalidPayload)
		}
	}

	return projectlotcreateduc.Command{
		MessageID: messageID,
		EventType: helper.ApplyTrim(req.EventType),
		Lot:       toParams(req, occurredAt),
	}, nil
}

func (Assembler) ToDTO(_ projectlotcreateduc.Command) response.Empty {
	return response.Empty{}
}

func toParams(req request.ProjectLotCreated, occurredAt time.Time) lotview.NewParams {
	d := req.Data

	return lotview.NewParams{
		LotID:        helper.ParseUUID(d.LotID),
		Code:         d.Code,
		Variety:      d.Variety,
		Process:      lotview.ParseProcess(d.Process),
		HarvestDate:  helper.ParseDate(d.HarvestDate),
		FarmID:       helper.ParseUUID(d.Farm.ID),
		FarmName:     d.Farm.Name,
		FarmAltitude: d.Farm.Altitude,
		ProducerID:   helper.ParseUUID(d.Producer.ID),
		ProducerName: d.Producer.Name,
		CustodianID:  helper.ParseUUID(d.CustodianID),
		CurrentStage: lotview.ParseStage(d.Stage),
		Status:       lotview.ParseStatus(d.Status),
		Version:      req.Version,
		CreatedAt:    occurredAt,
	}
}
