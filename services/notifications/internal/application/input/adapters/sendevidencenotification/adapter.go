package sendevidencenotification

import (
	"context"

	"rutadeorigen/notifications/internal/application/input/dto/request"
	"rutadeorigen/notifications/internal/application/input/ports"
	sendnotificationuc "rutadeorigen/notifications/internal/application/usecases/sendevidencenotification"
	"rutadeorigen/notifications/internal/domain/notification"
	"rutadeorigen/notifications/internal/platform/exception"
	"rutadeorigen/notifications/internal/platform/helper"
)

const (
	EvidenceProcessedEventType = "evidence.processed"
	EvidenceRejectedEventType  = "evidence.rejected"
)

var kindByEventType = map[string]notification.Kind{
	EvidenceProcessedEventType: notification.KindEvidenceProcessed,
	EvidenceRejectedEventType:  notification.KindEvidenceRejected,
}

var _ ports.SendEvidenceNotificationPort = (*Adapter)(nil)

type Adapter struct {
	useCase sendnotificationuc.UseCase
}

func New(useCase sendnotificationuc.UseCase) *Adapter {
	return &Adapter{useCase: useCase}
}

func (a *Adapter) Execute(ctx context.Context, input request.SendEvidenceNotification) error {
	eventType := helper.ApplyTrim(input.EventType)
	kind, known := kindByEventType[eventType]
	messageID := helper.ParseUUID(input.MessageID)
	evidenceID := helper.ParseUUID(input.EvidenceID)
	lotID := helper.ParseUUID(input.LotID)

	if !known || helper.IsDefaultUUID(messageID) || helper.IsDefaultUUID(evidenceID) || helper.IsDefaultUUID(lotID) {
		return exception.New(exception.CodeEventInvalidPayload)
	}

	return a.useCase.Execute(ctx, sendnotificationuc.Command{
		MessageID:      messageID,
		EventType:      eventType,
		Kind:           kind,
		EvidenceID:     evidenceID,
		LotID:          lotID,
		LotCode:        helper.ApplyTrim(input.LotCode),
		Stage:          helper.ApplyTrim(input.Stage),
		FileName:       helper.ApplyTrim(input.FileName),
		Reason:         helper.ApplyTrim(input.Reason),
		RecipientName:  helper.ApplyTrim(input.UploaderName),
		RecipientEmail: helper.ApplyTrim(input.UploaderEmail),
	})
}
