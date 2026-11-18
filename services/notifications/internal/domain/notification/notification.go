package notification

import (
	"time"

	"github.com/google/uuid"

	"rutadeorigen/notifications/internal/platform/exception"
	"rutadeorigen/notifications/internal/platform/helper"
)

const maxLastErrorLength = 500

type Notification struct {
	id         uuid.UUID
	evidenceID uuid.UUID
	kind       Kind
	recipient  Recipient
	status     Status
	attempts   int
	lastError  string
	createdAt  time.Time
	sentAt     time.Time
}

var defaultNotification = Notification{
	id:         helper.DefaultUUID(),
	evidenceID: helper.DefaultUUID(),
	kind:       KindUnknown,
	recipient:  DefaultRecipient(),
	status:     StatusUnknown,
	attempts:   helper.DefaultInt(),
	lastError:  helper.EmptyString,
	createdAt:  helper.DefaultDate(),
	sentAt:     helper.DefaultDate(),
}

func Default() *Notification {
	n := defaultNotification
	return &n
}

type NewParams struct {
	ID         uuid.UUID
	EvidenceID uuid.UUID
	Kind       Kind
	Recipient  Recipient
	CreatedAt  time.Time
}

func New(p NewParams) (*Notification, error) {
	if helper.IsDefaultUUID(p.ID) || helper.IsDefaultUUID(p.EvidenceID) ||
		!p.Kind.IsValid() || helper.IsDefaultDate(p.CreatedAt) {
		return Default(), exception.New(exception.CodeEventInvalidPayload)
	}
	if p.Recipient.IsDefault() {
		return Default(), exception.New(exception.CodeInvalidRecipient)
	}

	return &Notification{
		id:         p.ID,
		evidenceID: p.EvidenceID,
		kind:       p.Kind,
		recipient:  p.Recipient,
		status:     StatusPending,
		attempts:   helper.DefaultInt(),
		lastError:  helper.EmptyString,
		createdAt:  p.CreatedAt.UTC(),
		sentAt:     helper.DefaultDate(),
	}, nil
}

type RestoreParams struct {
	ID         uuid.UUID
	EvidenceID uuid.UUID
	Kind       Kind
	Recipient  Recipient
	Status     Status
	Attempts   int
	LastError  string
	CreatedAt  time.Time
	SentAt     time.Time
}

func Restore(p RestoreParams) *Notification {
	return &Notification{
		id:         p.ID,
		evidenceID: p.EvidenceID,
		kind:       ParseKind(string(p.Kind)),
		recipient:  p.Recipient,
		status:     ParseStatus(string(p.Status)),
		attempts:   p.Attempts,
		lastError:  p.LastError,
		createdAt:  p.CreatedAt.UTC(),
		sentAt:     p.SentAt.UTC(),
	}
}

func (n *Notification) RecordFailedAttempt(reason string) (*Notification, error) {
	if n.status != StatusPending {
		return Default(), exception.New(exception.CodeInvalidStatusTransition)
	}

	updated := *n
	updated.attempts++
	updated.lastError = limitReason(reason)
	return &updated, nil
}

func (n *Notification) MarkSent(at time.Time) (*Notification, error) {
	if n.status != StatusPending {
		return Default(), exception.New(exception.CodeInvalidStatusTransition)
	}

	updated := *n
	updated.status = StatusSent
	updated.attempts++
	updated.lastError = helper.EmptyString
	updated.sentAt = at.UTC()
	return &updated, nil
}

func (n *Notification) MarkFailed(reason string) (*Notification, error) {
	if n.status != StatusPending {
		return Default(), exception.New(exception.CodeInvalidStatusTransition)
	}

	updated := *n
	updated.status = StatusFailed
	updated.attempts++
	updated.lastError = limitReason(reason)
	return &updated, nil
}

func limitReason(reason string) string {
	trimmed := helper.ApplyTrim(reason)
	runes := []rune(trimmed)
	if len(runes) <= maxLastErrorLength {
		return trimmed
	}
	return string(runes[:maxLastErrorLength])
}

func (n *Notification) ID() uuid.UUID         { return n.id }
func (n *Notification) EvidenceID() uuid.UUID { return n.evidenceID }
func (n *Notification) Kind() Kind            { return n.kind }
func (n *Notification) Recipient() Recipient  { return n.recipient }
func (n *Notification) Status() Status        { return n.status }
func (n *Notification) Attempts() int         { return n.attempts }
func (n *Notification) LastError() string     { return n.lastError }
func (n *Notification) CreatedAt() time.Time  { return n.createdAt }
func (n *Notification) SentAt() time.Time     { return n.sentAt }

func (n *Notification) IsPending() bool { return n.status == StatusPending }
func (n *Notification) IsSent() bool    { return n.status == StatusSent }
func (n *Notification) IsFailed() bool  { return n.status == StatusFailed }

func (n *Notification) IsDefault() bool {
	return n == nil || helper.IsDefaultUUID(n.id)
}
