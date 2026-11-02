package postgres

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"rutadeorigen/processor/internal/application/output/ports"
)

var _ ports.EventPublisher = (*OutboxPublisher)(nil)

const insertOutbox = `
	INSERT INTO outbox (id, aggregate_type, aggregate_id, event_type, version, payload, created_at)
	VALUES ($1, $2, $3, $4, $5, $6, $7)`

type OutboxPublisher struct{ pool *pgxpool.Pool }

func NewOutboxPublisher(pool *pgxpool.Pool) *OutboxPublisher { return &OutboxPublisher{pool: pool} }

type envelope struct {
	ID            uuid.UUID `json:"id"`
	Type          string    `json:"type"`
	Version       int       `json:"version"`
	OccurredAt    time.Time `json:"occurred_at"`
	ActorID       uuid.UUID `json:"actor_id"`
	ActorRole     string    `json:"actor_role"`
	CorrelationID string    `json:"correlation_id"`
	Data          any       `json:"data"`
}

func (p *OutboxPublisher) Publish(ctx context.Context, event ports.Event) error {
	body, err := json.Marshal(envelope{
		ID:            event.ID,
		Type:          string(event.Type),
		Version:       event.Version,
		OccurredAt:    event.OccurredAt,
		ActorID:       event.ActorID,
		ActorRole:     event.ActorRole,
		CorrelationID: event.CorrelationID,
		Data:          event.Payload,
	})
	if err != nil {
		return internalError("marshal event", err)
	}

	_, err = conn(ctx, p.pool).Exec(ctx, insertOutbox, event.ID, event.AggregateType,
		event.AggregateID, event.Type, event.Version, body, event.OccurredAt)
	if err != nil {
		return internalError("insert outbox", err)
	}
	return nil
}
