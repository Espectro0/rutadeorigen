package projectlotstage

import (
	"rutadeorigen/qr/internal/application/input/assembler"
	"rutadeorigen/qr/internal/application/input/dto/request"
	"rutadeorigen/qr/internal/application/input/dto/response"
	projectlotstageuc "rutadeorigen/qr/internal/application/usecases/projectlotstage"
	"rutadeorigen/qr/internal/domain/lotview"
	"rutadeorigen/qr/internal/domain/stageview"
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

	eventDate := helper.ParseDate(req.Data.EventDate)
	if helper.IsDefaultDate(eventDate) {
		return projectlotstageuc.Command{}, exception.New(exception.CodeEventInvalidPayload)
	}

	return projectlotstageuc.Command{
		MessageID:  messageID,
		EventType:  helper.ApplyTrim(req.EventType),
		LotID:      lotID,
		Stage:      lotview.ParseStage(req.Data.Stage),
		Version:    req.Version,
		OccurredAt: occurredAt,
		EventDate:  eventDate,
		Details: stageview.Details{
			DryingMethod:         req.Data.Details.DryingMethod,
			DryingDays:           req.Data.Details.DryingDays,
			RoastLevel:           req.Data.Details.RoastLevel,
			RoastDurationMinutes: req.Data.Details.DurationMinutes,
			Grind:                req.Data.Details.Grind,
			PackageWeightGrams:   req.Data.Details.PackageWeightG,
			BestBefore:           req.Data.Details.BestBefore,
		},
	}, nil
}

func (Assembler) ToDTO(_ projectlotstageuc.Command) response.Empty {
	return response.Empty{}
}
