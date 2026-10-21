package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	libmessaging "rutadeorigen/lib/messaging"
)

type BrokerPublisher interface {
	Publish(ctx context.Context, m libmessaging.Message) error
}

type OutboxRelay struct {
	pool      *pgxpool.Pool
	publisher BrokerPublisher
	log       *slog.Logger
}

func NewOutboxRelay(pool *pgxpool.Pool, publisher BrokerPublisher, log *slog.Logger) *OutboxRelay {
	return &OutboxRelay{pool: pool, publisher: publisher, log: log}
}

type outboxRow struct {
	ID        uuid.UUID `db:"id"`
	EventType string    `db:"event_type"`
	Version   int       `db:"version"`
	Payload   []byte    `db:"payload"`
	CreatedAt time.Time `db:"created_at"`
}

func (r outboxRow) toMessage() libmessaging.Message {
	var meta struct {
		CorrelationID string `json:"correlation_id"`
	}
	_ = json.Unmarshal(r.Payload, &meta)
	return libmessaging.Message{
		ID:            r.ID.String(),
		Type:          r.EventType,
		CorrelationID: meta.CorrelationID,
		Version:       r.Version,
		OccurredAt:    r.CreatedAt,
		Body:          r.Payload,
	}
}

const claimOutbox = `
	SELECT id, event_type, version, payload, created_at FROM outbox
	WHERE published_at IS NULL ORDER BY created_at
	LIMIT 50 FOR UPDATE SKIP LOCKED`

func (r *OutboxRelay) Run(ctx context.Context) {
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if err := r.flush(ctx); err != nil {
				r.log.ErrorContext(ctx, "outbox relay failed", "err", err)
			}
		}
	}
}

func (r *OutboxRelay) flush(ctx context.Context) error {
	err := pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, claimOutbox)
		if err != nil {
			return fmt.Errorf("claim outbox: %w", err)
		}
		pending, err := pgx.CollectRows(rows, pgx.RowToStructByName[outboxRow])
		if err != nil {
			return fmt.Errorf("scan outbox: %w", err)
		}
		for _, row := range pending {
			if err := r.publisher.Publish(ctx, row.toMessage()); err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, `UPDATE outbox SET published_at = now() WHERE id = $1`, row.ID); err != nil {
				return fmt.Errorf("mark outbox %s as published: %w", row.ID, err)
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("outbox relay: %w", err)
	}
	return nil
}
