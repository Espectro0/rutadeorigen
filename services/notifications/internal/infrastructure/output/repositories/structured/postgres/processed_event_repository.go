package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"rutadeorigen/notifications/internal/application/output/ports"
)

var _ ports.ProcessedEventRepository = (*ProcessedEventRepository)(nil)

const (
	selectProcessedEventExists = `
	SELECT EXISTS (SELECT 1 FROM processed_event WHERE message_id = $1)`

	insertProcessedEvent = `
	INSERT INTO processed_event (message_id, event_type, processed_at)
	VALUES ($1, $2, NOW())
	ON CONFLICT (message_id) DO NOTHING`
)

type ProcessedEventRepository struct{ pool *pgxpool.Pool }

func NewProcessedEventRepository(pool *pgxpool.Pool) *ProcessedEventRepository {
	return &ProcessedEventRepository{pool: pool}
}

func (r *ProcessedEventRepository) Exists(ctx context.Context, messageID uuid.UUID) (bool, error) {
	var exists bool
	if err := conn(ctx, r.pool).QueryRow(ctx, selectProcessedEventExists, messageID).Scan(&exists); err != nil {
		return false, internalError("check processed event", err)
	}
	return exists, nil
}

func (r *ProcessedEventRepository) Register(ctx context.Context, messageID uuid.UUID, eventType string) (bool, error) {
	tag, err := conn(ctx, r.pool).Exec(ctx, insertProcessedEvent, messageID, eventType)
	if err != nil {
		return false, internalError("register processed event", err)
	}
	return tag.RowsAffected() == 1, nil
}
