package notification_test

import (
	"testing"

	"rutadeorigen/notifications/internal/domain/notification"
)

func TestParseKind(t *testing.T) {
	tests := map[string]notification.Kind{
		"evidence_processed":  notification.KindEvidenceProcessed,
		" Evidence_Rejected ": notification.KindEvidenceRejected,
		"":                    notification.KindUnknown,
		"evidence.processed":  notification.KindUnknown,
		"something else":      notification.KindUnknown,
	}

	for input, want := range tests {
		if got := notification.ParseKind(input); got != want {
			t.Errorf("ParseKind(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestKindValidity(t *testing.T) {
	if !notification.KindEvidenceProcessed.IsValid() || !notification.KindEvidenceRejected.IsValid() {
		t.Error("the evidence kinds must be valid")
	}
	if notification.KindUnknown.IsValid() || notification.Kind("weird").IsValid() {
		t.Error("unknown kinds must be invalid")
	}
	if notification.KindEvidenceRejected.String() != "evidence_rejected" {
		t.Error("String must return the raw value")
	}
}

func TestParseStatus(t *testing.T) {
	tests := map[string]notification.Status{
		"pending":  notification.StatusPending,
		" SENT ":   notification.StatusSent,
		"failed":   notification.StatusFailed,
		"":         notification.StatusUnknown,
		"archived": notification.StatusUnknown,
	}

	for input, want := range tests {
		if got := notification.ParseStatus(input); got != want {
			t.Errorf("ParseStatus(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestStatusFinality(t *testing.T) {
	if notification.StatusPending.IsFinal() || notification.StatusUnknown.IsFinal() {
		t.Error("pending and unknown statuses are not final")
	}
	if !notification.StatusSent.IsFinal() || !notification.StatusFailed.IsFinal() {
		t.Error("sent and failed statuses are final")
	}
	if notification.StatusUnknown.IsValid() || !notification.StatusPending.IsValid() {
		t.Error("only the known statuses are valid")
	}
	if notification.StatusSent.String() != "sent" {
		t.Error("String must return the raw value")
	}
}
