package ports

import (
	"context"

	"rutadeorigen/notifications/internal/domain/notification"
)

type NotificationLogRepository interface {
	Reserve(ctx context.Context, candidate *notification.Notification) (*notification.Notification, error)
	Update(ctx context.Context, updated *notification.Notification) error
}
