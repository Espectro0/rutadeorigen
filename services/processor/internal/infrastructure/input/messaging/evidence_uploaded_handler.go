package messaging

import (
	"context"
	"encoding/json"

	libmessaging "rutadeorigen/lib/messaging"
	"rutadeorigen/lib/requestcontext"
	"rutadeorigen/processor/internal/application/input/dto/request"
	"rutadeorigen/processor/internal/application/input/ports"
	"rutadeorigen/processor/internal/platform/exception"
)

const EvidenceUploadedType = "evidence.uploaded"

type envelope struct {
	ID            string               `json:"id"`
	Type          string               `json:"type"`
	CorrelationID string               `json:"correlation_id"`
	Data          evidenceUploadedData `json:"data"`
}

type evidenceUploadedData struct {
	EvidenceID    string   `json:"evidence_id"`
	LotID         string   `json:"lot_id"`
	QuarantineKey string   `json:"quarantine_key"`
	SizeBytes     int64    `json:"size_bytes"`
	Uploader      uploader `json:"uploader"`
}

type uploader struct {
	ActorID string `json:"actor_id"`
	Role    string `json:"role"`
}

type EvidenceUploadedHandler struct {
	process ports.ProcessEvidencePort
}

func NewEvidenceUploadedHandler(process ports.ProcessEvidencePort) *EvidenceUploadedHandler {
	return &EvidenceUploadedHandler{process: process}
}

func (h *EvidenceUploadedHandler) Handle(ctx context.Context, d libmessaging.Delivery) error {
	if d.Type != EvidenceUploadedType {
		return nil
	}

	var env envelope
	if err := json.Unmarshal(d.Body, &env); err != nil {
		return libmessaging.Permanent(exception.New(exception.CodeEventInvalidPayload).WithCause(err))
	}

	if correlationID := firstNonEmpty(d.CorrelationID, env.CorrelationID); correlationID != "" {
		ctx = requestcontext.WithCorrelationID(ctx, correlationID)
	}

	err := h.process.Execute(ctx, request.ProcessEvidence{
		MessageID:       firstNonEmpty(d.ID, env.ID),
		EventType:       d.Type,
		EvidenceID:      env.Data.EvidenceID,
		LotID:           env.Data.LotID,
		QuarantineKey:   env.Data.QuarantineKey,
		SizeBytes:       env.Data.SizeBytes,
		UploaderActorID: env.Data.Uploader.ActorID,
		UploaderRole:    env.Data.Uploader.Role,
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
