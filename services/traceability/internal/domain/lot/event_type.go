package lot

import (
	"strings"

	"rutadeorigen/traceability/internal/platform/helper"
)

type EventType string

const (
	EventTypeUnknown    EventType = "unknown"
	EventTypeStage      EventType = "stage"
	EventTypeTransfer   EventType = "transfer"
	EventTypeCorrection EventType = "correction"
)

func ParseEventType(value string) EventType {
	switch v := EventType(strings.ToLower(helper.ApplyTrim(value))); v {
	case EventTypeStage, EventTypeTransfer, EventTypeCorrection:
		return v
	default:
		return EventTypeUnknown
	}
}

func (v EventType) IsValid() bool {
	return ParseEventType(string(v)) != EventTypeUnknown
}

func (v EventType) String() string {
	return string(v)
}
