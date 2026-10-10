package lot

import (
	"strings"

	"rutadeorigen/traceability/internal/platform/helper"
)

type Stage string

const (
	StageUnknown   Stage = "unknown"
	StageNone      Stage = "none"
	StageFarm      Stage = "farm"
	StageRoasting  Stage = "roasting"
	StagePackaging Stage = "packaging"
)

var stageOrder = map[Stage]int{
	StageNone:      0,
	StageFarm:      1,
	StageRoasting:  2,
	StagePackaging: 3,
}

func ParseStage(value string) Stage {
	s := Stage(strings.ToLower(helper.ApplyTrim(value)))
	if _, ok := stageOrder[s]; !ok {
		return StageUnknown
	}
	return s
}

func (s Stage) IsValid() bool {
	return ParseStage(string(s)) != StageUnknown
}

func (s Stage) CanFollow(current Stage) bool {
	if !s.IsValid() || !current.IsValid() || s == StageNone {
		return false
	}
	return stageOrder[s] == stageOrder[current]+1
}

func (s Stage) Next() Stage {
	if !s.IsValid() {
		return StageUnknown
	}
	for stage, order := range stageOrder {
		if order == stageOrder[s]+1 {
			return stage
		}
	}
	return StageUnknown
}

func (s Stage) String() string {
	return string(s)
}
