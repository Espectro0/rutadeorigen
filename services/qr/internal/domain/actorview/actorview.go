package actorview

import (
	"time"

	"github.com/google/uuid"

	"rutadeorigen/qr/internal/platform/exception"
	"rutadeorigen/qr/internal/platform/helper"
)

type ActorView struct {
	actorID   uuid.UUID
	userID    string
	role      Role
	createdAt time.Time
}

var defaultActorView = ActorView{
	actorID:   helper.DefaultUUID(),
	userID:    helper.EmptyString,
	role:      RoleUnknown,
	createdAt: helper.DefaultDate(),
}

func Default() *ActorView {
	v := defaultActorView
	return &v
}

type NewParams struct {
	ActorID   uuid.UUID
	UserID    string
	Role      Role
	CreatedAt time.Time
}

func New(p NewParams) (*ActorView, error) {
	userID := helper.ApplyTrim(p.UserID)

	if helper.IsDefaultUUID(p.ActorID) {
		return Default(), exception.New(exception.CodeActorViewInvalidID)
	}
	if helper.IsEmpty(userID) {
		return Default(), exception.New(exception.CodeActorViewInvalidUser)
	}
	if !p.Role.IsValid() {
		return Default(), exception.New(exception.CodeActorViewInvalidRole)
	}

	createdAt := time.Now().UTC()
	if !helper.IsDefaultDate(p.CreatedAt) {
		createdAt = p.CreatedAt.UTC()
	}

	return &ActorView{
		actorID:   p.ActorID,
		userID:    userID,
		role:      p.Role,
		createdAt: createdAt,
	}, nil
}

type RestoreParams struct {
	ActorID   uuid.UUID
	UserID    string
	Role      Role
	CreatedAt time.Time
}

func Restore(p RestoreParams) *ActorView {
	return &ActorView{
		actorID:   p.ActorID,
		userID:    p.UserID,
		role:      p.Role,
		createdAt: p.CreatedAt,
	}
}

func (v *ActorView) ActorID() uuid.UUID   { return v.actorID }
func (v *ActorView) UserID() string       { return v.userID }
func (v *ActorView) Role() Role           { return v.role }
func (v *ActorView) CreatedAt() time.Time { return v.createdAt }
func (v *ActorView) IsDefault() bool      { return helper.IsDefaultUUID(v.actorID) }
func (v *ActorView) IsOrganization() bool { return v.role == RoleOrganization }
