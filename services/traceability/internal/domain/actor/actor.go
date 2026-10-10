package actor

import (
	"time"

	"github.com/google/uuid"

	"rutadeorigen/traceability/internal/platform/helper"
)

const (
	nameMinLength = 3
	nameMaxLength = 100
)

type Actor struct {
	id        uuid.UUID
	userID    string
	actorType Type
	name      string
	status    Status
	createdAt time.Time
}

var defaultActor = Actor{
	id:        helper.DefaultUUID(),
	userID:    helper.EmptyString,
	actorType: TypeUnknown,
	name:      helper.EmptyString,
	status:    StatusUnknown,
	createdAt: helper.DefaultDate(),
}

func Default() *Actor {
	a := defaultActor
	return &a
}

func New(userID string, actorType Type, name string) (*Actor, error) {
	userID = helper.ApplyTrim(userID)
	name = helper.ApplyTrim(name)

	if helper.IsEmpty(userID) {
		return Default(), ErrInvalidUserID
	}
	if !actorType.IsValid() {
		return Default(), ErrInvalidType
	}
	if !helper.LengthIsValid(name, nameMinLength, nameMaxLength) {
		return Default(), ErrInvalidName
	}

	return &Actor{
		id:        uuid.New(),
		userID:    userID,
		actorType: actorType,
		name:      name,
		status:    StatusActive,
		createdAt: time.Now().UTC(),
	}, nil
}

func Restore(id uuid.UUID, userID string, actorType Type, name string,
	status Status, createdAt time.Time) *Actor {
	return &Actor{
		id:        id,
		userID:    userID,
		actorType: actorType,
		name:      name,
		status:    status,
		createdAt: createdAt,
	}
}

func (a *Actor) IsActive() bool   { return a.status == StatusActive }
func (a *Actor) IsProducer() bool { return a.actorType == TypeProducer }

func (a *Actor) ID() uuid.UUID        { return a.id }
func (a *Actor) UserID() string       { return a.userID }
func (a *Actor) Type() Type           { return a.actorType }
func (a *Actor) Name() string         { return a.name }
func (a *Actor) Status() Status       { return a.status }
func (a *Actor) CreatedAt() time.Time { return a.createdAt }

func (a *Actor) IsDefault() bool {
	return a == nil || helper.IsDefaultUUID(a.id)
}
