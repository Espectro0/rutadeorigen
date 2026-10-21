package postgres

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"rutadeorigen/traceability/internal/application/output/ports"
)

type OutboxPublisher struct{ pool *pgxpool.Pool }

func NewOutboxPublisher(pool *pgxpool.Pool) *OutboxPublisher { return &OutboxPublisher{pool: pool} }

var _ ports.EventPublisher = (*OutboxPublisher)(nil)

const insertOutbox = `
	INSERT INTO outbox (id, aggregate_type, aggregate_id, event_type, version, payload, created_at)
	VALUES ($1, $2, $3, $4, $5, $6, $7)`

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

func (p *OutboxPublisher) Publish(ctx context.Context, e ports.Event) error {
	body, err := json.Marshal(envelope{e.ID, string(e.Type), e.Version, e.OccurredAt,
		e.ActorID, e.ActorRole, e.CorrelationID, e.Payload})
	if err != nil {
		return internalError("marshal event", err)
	}
	_, err = conn(ctx, p.pool).Exec(ctx, insertOutbox, e.ID, e.AggregateType,
		e.AggregateID, e.Type, e.Version, body, e.OccurredAt)
	if err != nil {
		return internalError("insert outbox", err)
	}
	return nil
}
