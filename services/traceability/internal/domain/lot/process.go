package lot

import (
	"strings"

	"rutadeorigen/traceability/internal/platform/helper"
)

type Process string

const (
	ProcessUnknown Process = "unknown"
	ProcessWashed  Process = "washed"
	ProcessNatural Process = "natural"
	ProcessHoney   Process = "honey"
)

func ParseProcess(value string) Process {
	switch v := Process(strings.ToLower(helper.ApplyTrim(value))); v {
	case ProcessWashed, ProcessNatural, ProcessHoney:
		return v
	default:
		return ProcessUnknown
	}
}

func (v Process) IsValid() bool {
	return ParseProcess(string(v)) != ProcessUnknown
}

func (v Process) String() string {
	return string(v)
}
