package memory

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"rutadeorigen/notifications/internal/domain/notification"
)

func newNotification(t *testing.T, evidenceID uuid.UUID, kind notification.Kind) *notification.Notification {
	t.Helper()

	recipient, err := notification.NewRecipient("Maria", "maria@example.com")
	if err != nil {
		t.Fatal(err)
	}
	n, err := notification.New(notification.NewParams{
		ID:         uuid.New(),
		EvidenceID: evidenceID,
		Kind:       kind,
		Recipient:  recipient,
		CreatedAt:  time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}
	return n
}

func TestReserveReturnsTheExistingNotificationForTheSameEvidenceAndKind(t *testing.T) {
	repository := NewNotificationLogRepository()
	evidenceID := uuid.New()
	first := newNotification(t, evidenceID, notification.KindEvidenceProcessed)
	second := newNotification(t, evidenceID, notification.KindEvidenceProcessed)

	got, err := repository.Reserve(context.Background(), first)
	if err != nil || got.ID() != first.ID() {
		t.Fatalf("first reserve returned %v %v", got, err)
	}

	got, err = repository.Reserve(context.Background(), second)
	if err != nil || got.ID() != first.ID() {
		t.Errorf("second reserve must return the first notification, got %v %v", got, err)
	}
}

func TestReserveSeparatesKinds(t *testing.T) {
	repository := NewNotificationLogRepository()
	evidenceID := uuid.New()
	processed := newNotification(t, evidenceID, notification.KindEvidenceProcessed)
	rejected := newNotification(t, evidenceID, notification.KindEvidenceRejected)

	repository.Reserve(context.Background(), processed)
	got, _ := repository.Reserve(context.Background(), rejected)

	if got.ID() != rejected.ID() {
		t.Error("a different kind must create a new notification")
	}
}

func TestUpdatePersistsTheChange(t *testing.T) {
	repository := NewNotificationLogRepository()
	pending := newNotification(t, uuid.New(), notification.KindEvidenceProcessed)
	repository.Reserve(context.Background(), pending)

	sent, err := pending.MarkSent(time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := repository.Update(context.Background(), sent); err != nil {
		t.Fatal(err)
	}

	got, _ := repository.Reserve(context.Background(), newNotification(t, pending.EvidenceID(), pending.Kind()))
	if !got.IsSent() {
		t.Error("the stored notification must be sent")
	}
}

func TestUpdateFailsForUnknownNotification(t *testing.T) {
	err := NewNotificationLogRepository().Update(context.Background(), newNotification(t, uuid.New(), notification.KindEvidenceProcessed))

	if err == nil {
		t.Error("want an error for an unknown notification")
	}
}
