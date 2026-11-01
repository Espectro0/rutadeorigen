package projectactorregistered

import (
	"rutadeorigen/qr/internal/application/input/assembler"
	"rutadeorigen/qr/internal/application/input/dto/request"
	"rutadeorigen/qr/internal/application/input/dto/response"
	projectactorregistereduc "rutadeorigen/qr/internal/application/usecases/projectactorregistered"
	"rutadeorigen/qr/internal/domain/actorview"
	"rutadeorigen/qr/internal/platform/exception"
	"rutadeorigen/qr/internal/platform/helper"
)

var _ assembler.DTOAssembler[projectactorregistereduc.Command, request.ProjectActorRegistered, response.Empty] = Assembler{}

type Assembler struct{}

func (Assembler) ToDomain(req request.ProjectActorRegistered) (projectactorregistereduc.Command, error) {
	messageID := helper.ParseUUID(req.MessageID)
	if helper.IsDefaultUUID(messageID) {
		return projectactorregistereduc.Command{}, exception.New(exception.CodeEventInvalidPayload)
	}

	actorID := helper.ParseUUID(req.Data.ActorID)
	if helper.IsDefaultUUID(actorID) {
		return projectactorregistereduc.Command{}, exception.New(exception.CodeEventInvalidPayload)
	}

	occurredAt := helper.DefaultDate()
	if !helper.IsEmpty(req.OccurredAt) {
		occurredAt = helper.ParseDateTime(req.OccurredAt)
		if helper.IsDefaultDate(occurredAt) {
			return projectactorregistereduc.Command{}, exception.New(exception.CodeEventInvalidPayload)
		}
	}

	return projectactorregistereduc.Command{
		MessageID:  messageID,
		EventType:  helper.ApplyTrim(req.EventType),
		ActorID:    actorID,
		UserID:     helper.ApplyTrim(req.Data.UserID),
		Role:       actorview.ParseRole(req.Data.Type),
		OccurredAt: occurredAt,
	}, nil
}

func (Assembler) ToDTO(_ projectactorregistereduc.Command) response.Empty {
	return response.Empty{}
}
