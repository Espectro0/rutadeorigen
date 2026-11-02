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
	LotTransferredType     = "lot.transferred"
	LotVoidedType          = "lot.voided"
	ActorRegisteredType    = "actor.registered"
	EvidenceProcessedType  = "evidence.processed"
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
	created         ports.ProjectLotCreatedPort
	staged          ports.ProjectLotStagePort
	actorRegistered ports.ProjectActorRegisteredPort
	transferred     ports.ProjectLotTransferredPort
	voided          ports.ProjectLotVoidedPort
	evidence        ports.ProjectEvidenceProcessedPort
}

func NewLotEventsHandler(created ports.ProjectLotCreatedPort, staged ports.ProjectLotStagePort) *LotEventsHandler {
	return &LotEventsHandler{created: created, staged: staged}
}

func (h *LotEventsHandler) WithActorRegistered(port ports.ProjectActorRegisteredPort) *LotEventsHandler {
	h.actorRegistered = port
	return h
}

func (h *LotEventsHandler) WithLotTransferred(port ports.ProjectLotTransferredPort) *LotEventsHandler {
	h.transferred = port
	return h
}

func (h *LotEventsHandler) WithLotVoided(port ports.ProjectLotVoidedPort) *LotEventsHandler {
	h.voided = port
	return h
}

func (h *LotEventsHandler) WithEvidenceProcessed(port ports.ProjectEvidenceProcessedPort) *LotEventsHandler {
	h.evidence = port
	return h
}

func (h *LotEventsHandler) handles(eventType string) bool {
	switch eventType {
	case LotCreatedType, LotStageRegisteredType:
		return true
	case ActorRegisteredType:
		return h.actorRegistered != nil
	case LotTransferredType:
		return h.transferred != nil
	case LotVoidedType:
		return h.voided != nil
	case EvidenceProcessedType:
		return h.evidence != nil
	default:
		return false
	}
}

func (h *LotEventsHandler) Handle(ctx context.Context, d libmessaging.Delivery) error {
	var env envelope
	if err := json.Unmarshal(d.Body, &env); err != nil {
		return libmessaging.Permanent(exception.New(exception.CodeEventInvalidPayload).WithCause(err))
	}

	eventType := firstNonEmpty(d.Type, env.Type)
	if !h.handles(eventType) {
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
	case ActorRegisteredType:
		return h.projectActorRegistered(ctx, meta, env.Data)
	case LotTransferredType:
		return h.projectTransferred(ctx, meta, env.Data)
	case LotVoidedType:
		return h.projectVoided(ctx, meta, env.Data)
	case EvidenceProcessedType:
		return h.projectEvidenceProcessed(ctx, meta, env.Data)
	default:
		return h.projectStage(ctx, meta, env.Data)
	}
}

func (h *LotEventsHandler) projectActorRegistered(ctx context.Context, meta metadata, raw json.RawMessage) error {
	data, err := decodeData[request.ActorRegisteredData](raw)
	if err != nil {
		return err
	}

	_, err = h.actorRegistered.Execute(ctx, request.ProjectActorRegistered{
		MessageID:  meta.messageID,
		EventType:  meta.eventType,
		Version:    meta.version,
		OccurredAt: meta.occurredAt,
		Data:       data,
	})
	return classify(err)
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

func (h *LotEventsHandler) projectTransferred(ctx context.Context, meta metadata, raw json.RawMessage) error {
	data, err := decodeData[request.LotTransferredData](raw)
	if err != nil {
		return err
	}

	_, err = h.transferred.Execute(ctx, request.ProjectLotTransferred{
		MessageID:  meta.messageID,
		EventType:  meta.eventType,
		Version:    meta.version,
		OccurredAt: meta.occurredAt,
		Data:       data,
	})
	return classify(err)
}

func (h *LotEventsHandler) projectVoided(ctx context.Context, meta metadata, raw json.RawMessage) error {
	data, err := decodeData[request.LotVoidedData](raw)
	if err != nil {
		return err
	}

	_, err = h.voided.Execute(ctx, request.ProjectLotVoided{
		MessageID:  meta.messageID,
		EventType:  meta.eventType,
		Version:    meta.version,
		OccurredAt: meta.occurredAt,
		Data:       data,
	})
	return classify(err)
}

func (h *LotEventsHandler) projectEvidenceProcessed(ctx context.Context, meta metadata, raw json.RawMessage) error {
	data, err := decodeData[request.EvidenceProcessedData](raw)
	if err != nil {
		return err
	}

	_, err = h.evidence.Execute(ctx, request.ProjectEvidenceProcessed{
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
