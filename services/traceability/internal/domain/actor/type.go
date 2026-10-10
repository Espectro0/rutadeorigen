package actor

import (
	"strings"

	"rutadeorigen/traceability/internal/platform/helper"
)

type Type string

const (
	TypeUnknown      Type = "unknown"
	TypeOrganization Type = "organization"
	TypeProducer     Type = "producer"
	TypeRoaster      Type = "roaster"
	TypeBrand        Type = "brand"
)

func ParseType(value string) Type {
	switch v := Type(strings.ToLower(helper.ApplyTrim(value))); v {
	case TypeOrganization, TypeProducer, TypeRoaster, TypeBrand:
		return v
	default:
		return TypeUnknown
	}
}

func (v Type) IsValid() bool {
	return ParseType(string(v)) != TypeUnknown
}

func (v Type) String() string {
	return string(v)
}
