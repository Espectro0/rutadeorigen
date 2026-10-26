package messaging

import (
	"context"
	"encoding/json"
	"time"

	libmessaging "rutadeorigen/lib/messaging"
	"rutadeorigen/lib/requestcontext"
	"rutadeorigen/qr/internal/application/input/dto/request"
	"rutadeorigen/qr/internal/application/input/ports"
	"rutadeorigen/qr/internal/platform/exception"
)

const (
	LotCreatedType         = "lot.created"
	LotStageRegisteredType = "lot.stage_registered"
)

type envelope struct {
	ID            string          `json:"id"`
	Type          string          `json:"type"`
	Version       int             `json:"version"`
	OccurredAt    string          `json:"occurred_at"`
	CorrelationID string          `json:"correlation_id"`
	Data          json.RawMessage `json:"data"`
}

type metadata struct {
	messageID  string
	eventType  string
	version    int
	occurredAt string
}

type LotEventsHandler struct {
	created ports.ProjectLotCreatedPort
	staged  ports.ProjectLotStagePort
}

func NewLotEventsHandler(created ports.ProjectLotCreatedPort, staged ports.ProjectLotStagePort) *LotEventsHandler {
	return &LotEventsHandler{created: created, staged: staged}
}
func (h *LotEventsHandler) Handle(ctx context.Context, d libmessaging.Delivery) error {
	var env envelope
	if err := json.Unmarshal(d.Body, &env); err != nil {
		return libmessaging.Permanent(exception.New(exception.CodeEventInvalidPayload).WithCause(err))
	}

	eventType := firstNonEmpty(d.Type, env.Type)
	if eventType != LotCreatedType && eventType != LotStageRegisteredType {
		return nil
	}

	if correlationID := firstNonEmpty(d.CorrelationID, env.CorrelationID); correlationID != "" {
		ctx = requestcontext.WithCorrelationID(ctx, correlationID)
	}

	meta := metadata{
		messageID:  firstNonEmpty(d.ID, env.ID),
		eventType:  eventType,
		version:    firstPositive(d.Version, env.Version),
		occurredAt: occurredAt(d, env),
	}

	switch eventType {
	case LotCreatedType:
		return h.projectCreated(ctx, meta, env.Data)
	default:
		return h.projectStage(ctx, meta, env.Data)
	}
}

func (h *LotEventsHandler) projectCreated(ctx context.Context, meta metadata, raw json.RawMessage) error {
	data, err := decodeData[request.LotCreatedData](raw)
	if err != nil {
		return err
	}

	_, err = h.created.Execute(ctx, request.ProjectLotCreated{
		MessageID:  meta.messageID,
		EventType:  meta.eventType,
		Version:    meta.version,
		OccurredAt: meta.occurredAt,
		Data:       data,
	})
	return classify(err)
}

func (h *LotEventsHandler) projectStage(ctx context.Context, meta metadata, raw json.RawMessage) error {
	data, err := decodeData[request.LotStageData](raw)
	if err != nil {
		return err
	}

	_, err = h.staged.Execute(ctx, request.ProjectLotStage{
		MessageID:  meta.messageID,
		EventType:  meta.eventType,
		Version:    meta.version,
		OccurredAt: meta.occurredAt,
		Data:       data,
	})
	return classify(err)
}

func decodeData[T any](raw json.RawMessage) (T, error) {
	var data T
	if err := json.Unmarshal(raw, &data); err != nil {
		return data, libmessaging.Permanent(exception.New(exception.CodeEventInvalidPayload).WithCause(err))
	}
	return data, nil
}

func occurredAt(d libmessaging.Delivery, env envelope) string {
	if env.OccurredAt != "" {
		return env.OccurredAt
	}
	if !d.OccurredAt.IsZero() {
		return d.OccurredAt.UTC().Format(time.RFC3339)
	}
	return ""
}

func firstPositive(values ...int) int {
	for _, v := range values {
		if v > 0 {
			return v
		}
	}
	return 0
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
