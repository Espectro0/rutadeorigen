package notification

import "rutadeorigen/notifications/internal/platform/helper"

type Status string

const (
	StatusUnknown Status = "unknown"
	StatusPending Status = "pending"
	StatusSent    Status = "sent"
	StatusFailed  Status = "failed"
)

func ParseStatus(value string) Status {
	switch status := Status(helper.ApplyLower(helper.ApplyTrim(value))); status {
	case StatusPending, StatusSent, StatusFailed:
		return status
	default:
		return StatusUnknown
	}
}

func (s Status) IsValid() bool {
	return ParseStatus(string(s)) != StatusUnknown
}

func (s Status) IsFinal() bool {
	return s == StatusSent || s == StatusFailed
}

func (s Status) String() string {
	return string(s)
}
