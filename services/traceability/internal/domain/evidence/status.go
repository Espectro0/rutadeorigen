package evidence

import (
	"strings"

	"rutadeorigen/traceability/internal/platform/helper"
)

type Status string

const (
	StatusUnknown       Status = "unknown"
	StatusPendingUpload Status = "pending_upload"
	StatusUploaded      Status = "uploaded"
	StatusProcessed     Status = "processed"
	StatusRejected      Status = "rejected"
)

func ParseStatus(value string) Status {
	switch s := Status(strings.ToLower(helper.ApplyTrim(value))); s {
	case StatusPendingUpload, StatusUploaded, StatusProcessed, StatusRejected:
		return s
	default:
		return StatusUnknown
	}
}

func (s Status) IsValid() bool { return ParseStatus(string(s)) != StatusUnknown }

func (s Status) IsFinal() bool { return s == StatusProcessed || s == StatusRejected }

func (s Status) String() string { return string(s) }
