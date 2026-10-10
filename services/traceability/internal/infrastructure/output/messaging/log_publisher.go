package messaging

import (
	"context"
	"log/slog"
	"time"

	"rutadeorigen/traceability/internal/application/output/ports"
)

var _ ports.EventPublisher = (*LogPublisher)(nil)

type LogPublisher struct {
	log *slog.Logger
}

func NewLogPublisher(log *slog.Logger) *LogPublisher {
	return &LogPublisher{log: log}
}

func (p *LogPublisher) Publish(ctx context.Context, event ports.Event) error {
	p.log.InfoContext(ctx, "event published",
		slog.String("event_id", event.ID.String()),
		slog.String("event_type", string(event.Type)),
		slog.String("aggregate_type", string(event.AggregateType)),
		slog.String("aggregate_id", event.AggregateID.String()),
		slog.Int("version", event.Version),
		slog.String("occurred_at", event.OccurredAt.Format(time.RFC3339)),
		slog.String("actor_id", event.ActorID.String()),
		slog.String("actor_role", event.ActorRole),
		slog.String("correlation_id", event.CorrelationID),
		slog.Any("payload", event.Payload),
	)
	return nil
}
