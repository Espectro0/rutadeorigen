package messaging

import (
	"context"
	"encoding/json"

	libmessaging "rutadeorigen/lib/messaging"
	"rutadeorigen/lib/requestcontext"
	"rutadeorigen/notifications/internal/application/input/dto/request"
	"rutadeorigen/notifications/internal/application/input/ports"
	"rutadeorigen/notifications/internal/platform/exception"
)

const (
	EvidenceProcessedType = "evidence.processed"
	EvidenceRejectedType  = "evidence.rejected"
)

type envelope struct {
	ID            string             `json:"id"`
	Type          string             `json:"type"`
	CorrelationID string             `json:"correlation_id"`
	Data          evidenceResultData `json:"data"`
}

type evidenceResultData struct {
	EvidenceID string   `json:"evidence_id"`
	LotID      string   `json:"lot_id"`
	LotCode    string   `json:"lot_code"`
	Stage      string   `json:"stage"`
	FileName   string   `json:"file_name"`
	Reason     string   `json:"reason"`
	Uploader   uploader `json:"uploader"`
}

type uploader struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type EvidenceResultHandler struct {
	send ports.SendEvidenceNotificationPort
}

func NewEvidenceResultHandler(send ports.SendEvidenceNotificationPort) *EvidenceResultHandler {
	return &EvidenceResultHandler{send: send}
}

func (h *EvidenceResultHandler) Handle(ctx context.Context, d libmessaging.Delivery) error {
	if d.Type != EvidenceProcessedType && d.Type != EvidenceRejectedType {
		return nil
	}

	var env envelope
	if err := json.Unmarshal(d.Body, &env); err != nil {
		return libmessaging.Permanent(exception.New(exception.CodeEventInvalidPayload).WithCause(err))
	}

	if correlationID := firstNonEmpty(d.CorrelationID, env.CorrelationID); correlationID != "" {
		ctx = requestcontext.WithCorrelationID(ctx, correlationID)
	}

	err := h.send.Execute(ctx, request.SendEvidenceNotification{
		MessageID:     firstNonEmpty(d.ID, env.ID),
		EventType:     d.Type,
		EvidenceID:    env.Data.EvidenceID,
		LotID:         env.Data.LotID,
		LotCode:       env.Data.LotCode,
		Stage:         env.Data.Stage,
		FileName:      env.Data.FileName,
		Reason:        env.Data.Reason,
		UploaderName:  env.Data.Uploader.Name,
		UploaderEmail: env.Data.Uploader.Email,
	})
	return classify(err)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
