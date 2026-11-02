package notification_test

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"rutadeorigen/notifications/internal/domain/notification"
	"rutadeorigen/notifications/internal/platform/exception"
)

var createdAt = time.Date(2026, time.November, 2, 15, 30, 0, 0, time.UTC)

func recipient(t *testing.T) notification.Recipient {
	t.Helper()

	r, err := notification.NewRecipient("Maria Lopez", "maria@example.com")
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func validParams(t *testing.T) notification.NewParams {
	t.Helper()

	return notification.NewParams{
		ID:         uuid.New(),
		EvidenceID: uuid.New(),
		Kind:       notification.KindEvidenceProcessed,
		Recipient:  recipient(t),
		CreatedAt:  createdAt,
	}
}

func pendingNotification(t *testing.T) *notification.Notification {
	t.Helper()

	n, err := notification.New(validParams(t))
	if err != nil {
		t.Fatal(err)
	}
	return n
}

func TestNewStartsPending(t *testing.T) {
	p := validParams(t)

	n, err := notification.New(p)

	if err != nil || n.IsDefault() {
		t.Fatalf("got %+v %v", n, err)
	}
	if n.ID() != p.ID || n.EvidenceID() != p.EvidenceID || n.Kind() != p.Kind || n.Recipient() != p.Recipient {
		t.Errorf("fields were not kept: %+v", n)
	}
	if !n.IsPending() || n.Attempts() != 0 || n.LastError() != "" || !n.SentAt().IsZero() {
		t.Errorf("a new notification must be pending with no attempts: %+v", n)
	}
	if !n.CreatedAt().Equal(createdAt) {
		t.Errorf("created at: got %v", n.CreatedAt())
	}
}

func TestNewRejectsInvalidParams(t *testing.T) {
	tests := map[string]struct {
		change func(*notification.NewParams)
		code   exception.Code
	}{
		"missing id":          {func(p *notification.NewParams) { p.ID = uuid.Nil }, exception.CodeEventInvalidPayload},
		"missing evidence id": {func(p *notification.NewParams) { p.EvidenceID = uuid.Nil }, exception.CodeEventInvalidPayload},
		"unknown kind":        {func(p *notification.NewParams) { p.Kind = notification.KindUnknown }, exception.CodeEventInvalidPayload},
		"missing created at":  {func(p *notification.NewParams) { p.CreatedAt = time.Time{} }, exception.CodeEventInvalidPayload},
		"default recipient":   {func(p *notification.NewParams) { p.Recipient = notification.DefaultRecipient() }, exception.CodeInvalidRecipient},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			p := validParams(t)
			tt.change(&p)

			n, err := notification.New(p)

			if !exception.HasCode(err, tt.code) {
				t.Errorf("error: got %v, want code %s", err, tt.code)
			}
			if !n.IsDefault() {
				t.Errorf("want the default notification, got %+v", n)
			}
		})
	}
}

func TestMarkSentFromPending(t *testing.T) {
	original := pendingNotification(t)
	sentAt := createdAt.Add(time.Minute)

	sent, err := original.MarkSent(sentAt)

	if err != nil || !sent.IsSent() || sent.Attempts() != 1 || sent.LastError() != "" || !sent.SentAt().Equal(sentAt) {
		t.Fatalf("got %+v %v", sent, err)
	}
	if !original.IsPending() || original.Attempts() != 0 {
		t.Error("the original must not change")
	}
}

func TestRecordFailedAttemptKeepsItPending(t *testing.T) {
	n := pendingNotification(t)

	once, err := n.RecordFailedAttempt("  resend timeout  ")
	if err != nil {
		t.Fatal(err)
	}
	twice, err := once.RecordFailedAttempt("resend 503")
	if err != nil {
		t.Fatal(err)
	}

	if !once.IsPending() || once.Attempts() != 1 || once.LastError() != "resend timeout" {
		t.Errorf("first attempt: %+v", once)
	}
	if !twice.IsPending() || twice.Attempts() != 2 || twice.LastError() != "resend 503" {
		t.Errorf("second attempt: %+v", twice)
	}
}

func TestMarkFailedFromPending(t *testing.T) {
	original := pendingNotification(t)

	failed, err := original.MarkFailed("resend rejected the recipient")

	if err != nil || !failed.IsFailed() || failed.Attempts() != 1 || failed.LastError() != "resend rejected the recipient" {
		t.Fatalf("got %+v %v", failed, err)
	}
	if !failed.SentAt().IsZero() {
		t.Error("a failed notification has no sent time")
	}
	if !original.IsPending() {
		t.Error("the original must not change")
	}
}

func TestLastErrorIsLimited(t *testing.T) {
	failed, err := pendingNotification(t).MarkFailed(strings.Repeat("é", 800))

	if err != nil {
		t.Fatal(err)
	}
	if got := len([]rune(failed.LastError())); got != 500 {
		t.Errorf("last error length: got %d, want 500", got)
	}
}

func TestFinalNotificationsCannotChange(t *testing.T) {
	sent, err := pendingNotification(t).MarkSent(createdAt)
	if err != nil {
		t.Fatal(err)
	}
	failed, err := pendingNotification(t).MarkFailed("rejected")
	if err != nil {
		t.Fatal(err)
	}

	for name, n := range map[string]*notification.Notification{"sent": sent, "failed": failed} {
		t.Run(name, func(t *testing.T) {
			_, errSent := n.MarkSent(createdAt)
			_, errFailed := n.MarkFailed("again")
			_, errAttempt := n.RecordFailedAttempt("again")

			for _, err := range []error{errSent, errFailed, errAttempt} {
				if !exception.HasCode(err, exception.CodeInvalidStatusTransition) {
					t.Errorf("want an invalid transition error, got %v", err)
				}
			}
		})
	}
}

func TestRestoreKeepsStoredState(t *testing.T) {
	id, evidenceID := uuid.New(), uuid.New()
	sentAt := createdAt.Add(time.Hour)

	n := notification.Restore(notification.RestoreParams{
		ID: id, EvidenceID: evidenceID, Kind: notification.KindEvidenceRejected, Recipient: recipient(t),
		Status: notification.StatusSent, Attempts: 3, LastError: "timeout", CreatedAt: createdAt, SentAt: sentAt,
	})

	if n.ID() != id || n.EvidenceID() != evidenceID || n.Kind() != notification.KindEvidenceRejected || !n.IsSent() {
		t.Errorf("unexpected %+v", n)
	}
	if n.Attempts() != 3 || n.LastError() != "timeout" || !n.CreatedAt().Equal(createdAt) || !n.SentAt().Equal(sentAt) {
		t.Errorf("unexpected %+v", n)
	}
}

func TestRestoreNormalizesUnknownValues(t *testing.T) {
	n := notification.Restore(notification.RestoreParams{Kind: "weird", Status: "weird"})

	if n.Kind() != notification.KindUnknown || n.Status() != notification.StatusUnknown {
		t.Errorf("got kind %q and status %q", n.Kind(), n.Status())
	}
}

func TestDefaultNotification(t *testing.T) {
	n := notification.Default()

	if !n.IsDefault() || n.Status() != notification.StatusUnknown || n.Kind() != notification.KindUnknown {
		t.Errorf("unexpected default %+v", n)
	}
	if !n.Recipient().IsDefault() || n.IsPending() || n.IsSent() || n.IsFailed() {
		t.Errorf("unexpected default %+v", n)
	}

	var missing *notification.Notification
	if !missing.IsDefault() {
		t.Error("a nil notification must report as default")
	}
}
