package memory

import (
	"context"
	"sync"

	"github.com/google/uuid"

	"rutadeorigen/notifications/internal/application/output/ports"
	"rutadeorigen/notifications/internal/domain/notification"
	"rutadeorigen/notifications/internal/platform/exception"
)

var _ ports.NotificationLogRepository = (*NotificationLogRepository)(nil)

type logKey struct {
	evidenceID uuid.UUID
	kind       notification.Kind
}

type NotificationLogRepository struct {
	mu   sync.Mutex
	byID map[uuid.UUID]*notification.Notification
	keys map[logKey]uuid.UUID
}

func NewNotificationLogRepository() *NotificationLogRepository {
	return &NotificationLogRepository{
		byID: make(map[uuid.UUID]*notification.Notification),
		keys: make(map[logKey]uuid.UUID),
	}
}

func (r *NotificationLogRepository) Reserve(_ context.Context, candidate *notification.Notification) (*notification.Notification, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := logKey{evidenceID: candidate.EvidenceID(), kind: candidate.Kind()}
	if id, found := r.keys[key]; found {
		return r.byID[id], nil
	}

	r.byID[candidate.ID()] = candidate
	r.keys[key] = candidate.ID()
	return candidate, nil
}

func (r *NotificationLogRepository) Update(_ context.Context, updated *notification.Notification) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, found := r.byID[updated.ID()]; !found {
		return exception.New(exception.CodeInternal).WithTechnical("update unknown notification")
	}
	r.byID[updated.ID()] = updated
	return nil
}
