package actorview

import (
	"strings"

	"rutadeorigen/qr/internal/platform/helper"
)

type Role string

const (
	RoleUnknown      Role = "unknown"
	RoleOrganization Role = "organization"
	RoleProducer     Role = "producer"
	RoleRoaster      Role = "roaster"
	RoleBrand        Role = "brand"
)

func ParseRole(value string) Role {
	switch v := Role(strings.ToLower(helper.ApplyTrim(value))); v {
	case RoleOrganization, RoleProducer, RoleRoaster, RoleBrand:
		return v
	default:
		return RoleUnknown
	}
}

func (v Role) IsValid() bool {
	return ParseRole(string(v)) != RoleUnknown
}

func (v Role) String() string {
	return string(v)
}
