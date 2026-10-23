package messaging

import (
	"context"
	"encoding/json"

	libmessaging "rutadeorigen/lib/messaging"
	"rutadeorigen/lib/requestcontext"
	"rutadeorigen/qr/internal/application/input/dto/request"
	"rutadeorigen/qr/internal/application/input/ports"
	"rutadeorigen/qr/internal/platform/exception"
)

const LotCreatedType = "lot.created"

type envelope struct {
	ID            string                 `json:"id"`
	Type          string                 `json:"type"`
	Version       int                    `json:"version"`
	OccurredAt    string                 `json:"occurred_at"`
	CorrelationID string                 `json:"correlation_id"`
	Data          request.LotCreatedData `json:"data"`
}

type LotCreatedHandler struct {
	project ports.ProjectLotCreatedPort
}

func NewLotCreatedHandler(project ports.ProjectLotCreatedPort) *LotCreatedHandler {
	return &LotCreatedHandler{project: project}
}

func (h *LotCreatedHandler) Handle(ctx context.Context, d libmessaging.Delivery) error {
	var env envelope
	if err := json.Unmarshal(d.Body, &env); err != nil {
		return libmessaging.Permanent(exception.New(exception.CodeEventInvalidPayload).WithCause(err))
	}

	eventType := firstNonEmpty(d.Type, env.Type)
	if eventType != LotCreatedType {
		return nil
	}

	if correlationID := firstNonEmpty(d.CorrelationID, env.CorrelationID); correlationID != "" {
		ctx = requestcontext.WithCorrelationID(ctx, correlationID)
	}

	version := d.Version
	if version == 0 {
		version = env.Version
	}

	occurredAt := env.OccurredAt
	if occurredAt == "" && !d.OccurredAt.IsZero() {
		occurredAt = d.OccurredAt.UTC().Format("2006-01-02T15:04:05Z07:00")
	}

	_, err := h.project.Execute(ctx, request.ProjectLotCreated{
		MessageID:  firstNonEmpty(d.ID, env.ID),
		EventType:  eventType,
		Version:    version,
		OccurredAt: occurredAt,
		Data:       env.Data,
	})
	return classify(err)
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
