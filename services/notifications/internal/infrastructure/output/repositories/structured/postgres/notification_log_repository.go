package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"rutadeorigen/notifications/internal/application/output/ports"
	"rutadeorigen/notifications/internal/domain/notification"
	"rutadeorigen/notifications/internal/platform/helper"
)

var _ ports.NotificationLogRepository = (*NotificationLogRepository)(nil)

const (
	insertNotificationLog = `
	INSERT INTO notification_log
		(id, evidence_id, kind, recipient_name, recipient_email, status, attempts, last_error, created_at, sent_at)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	ON CONFLICT (evidence_id, kind) DO NOTHING`

	selectNotificationLog = `
	SELECT id, evidence_id, kind, recipient_name, recipient_email, status, attempts, last_error, created_at, sent_at
	FROM notification_log
	WHERE evidence_id = $1 AND kind = $2`

	updateNotificationLog = `
	UPDATE notification_log
	SET status = $2, attempts = $3, last_error = $4, sent_at = $5
	WHERE id = $1`
)

type NotificationLogRepository struct{ pool *pgxpool.Pool }

func NewNotificationLogRepository(pool *pgxpool.Pool) *NotificationLogRepository {
	return &NotificationLogRepository{pool: pool}
}

func (r *NotificationLogRepository) Reserve(ctx context.Context, candidate *notification.Notification) (*notification.Notification, error) {
	db := conn(ctx, r.pool)

	_, err := db.Exec(ctx, insertNotificationLog,
		candidate.ID(), candidate.EvidenceID(), candidate.Kind().String(),
		candidate.Recipient().Name(), candidate.Recipient().Email(),
		candidate.Status().String(), candidate.Attempts(), candidate.LastError(),
		candidate.CreatedAt(), sentAtOrNil(candidate.SentAt()),
	)
	if err != nil {
		return notification.Default(), internalError("reserve notification", err)
	}

	var (
		id, evidenceID uuid.UUID
		kind, name     string
		email, status  string
		attempts       int
		lastError      string
		createdAt      time.Time
		sentAt         *time.Time
	)
	err = db.QueryRow(ctx, selectNotificationLog, candidate.EvidenceID(), candidate.Kind().String()).
		Scan(&id, &evidenceID, &kind, &name, &email, &status, &attempts, &lastError, &createdAt, &sentAt)
	if err != nil {
		return notification.Default(), internalError("load notification", err)
	}

	recipient, err := notification.NewRecipient(name, email)
	if err != nil {
		recipient = notification.DefaultRecipient()
	}

	return notification.Restore(notification.RestoreParams{
		ID:         id,
		EvidenceID: evidenceID,
		Kind:       notification.ParseKind(kind),
		Recipient:  recipient,
		Status:     notification.ParseStatus(status),
		Attempts:   attempts,
		LastError:  lastError,
		CreatedAt:  createdAt,
		SentAt:     helper.DateOrDefault(sentAt),
	}), nil
}

func (r *NotificationLogRepository) Update(ctx context.Context, updated *notification.Notification) error {
	tag, err := conn(ctx, r.pool).Exec(ctx, updateNotificationLog,
		updated.ID(), updated.Status().String(), updated.Attempts(), updated.LastError(), sentAtOrNil(updated.SentAt()),
	)
	if err != nil {
		return internalError("update notification", err)
	}
	if tag.RowsAffected() != 1 {
		return internalError("update unknown notification", nil)
	}
	return nil
}

func sentAtOrNil(value time.Time) *time.Time {
	if helper.IsDefaultDate(value) {
		return nil
	}
	return &value
}
