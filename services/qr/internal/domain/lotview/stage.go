package lotview

import (
	"strings"

	"rutadeorigen/qr/internal/platform/helper"
)

type Stage string

const (
	StageUnknown   Stage = "unknown"
	StageNone      Stage = "none"
	StageFarm      Stage = "farm"
	StageRoasting  Stage = "roasting"
	StagePackaging Stage = "packaging"
)

func ParseStage(value string) Stage {
	switch v := Stage(strings.ToLower(helper.ApplyTrim(value))); v {
	case StageNone, StageFarm, StageRoasting, StagePackaging:
		return v
	default:
		return StageUnknown
	}
}

func (v Stage) IsValid() bool {
	return ParseStage(string(v)) != StageUnknown
}

func (v Stage) String() string {
	return string(v)
}
