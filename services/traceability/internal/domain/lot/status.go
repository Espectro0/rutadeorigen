package lot

import (
	"strings"

	"rutadeorigen/traceability/internal/platform/helper"
)

type Status string

const (
	StatusUnknown Status = "unknown"
	StatusActive  Status = "active"
	StatusVoided  Status = "voided"
)

func ParseStatus(value string) Status {
	switch v := Status(strings.ToLower(helper.ApplyTrim(value))); v {
	case StatusActive, StatusVoided:
		return v
	default:
		return StatusUnknown
	}
}

func (v Status) IsValid() bool {
	return ParseStatus(string(v)) != StatusUnknown
}

func (v Status) String() string {
	return string(v)
}
