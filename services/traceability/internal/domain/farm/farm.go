package farm

import (
	"time"

	"github.com/google/uuid"

	"rutadeorigen/traceability/internal/platform/helper"
)

const (
	nameMinLength = 3
	nameMaxLength = 100
	minAltitude   = 0
	maxAltitude   = 3000
)

type Farm struct {
	id         uuid.UUID
	producerID uuid.UUID
	name       string
	altitude   int
	createdAt  time.Time
}

var defaultFarm = Farm{
	id:         helper.DefaultUUID(),
	producerID: helper.DefaultUUID(),
	name:       helper.EmptyString,
	altitude:   helper.DefaultInt(),
	createdAt:  helper.DefaultDate(),
}

func Default() *Farm {
	f := defaultFarm
	return &f
}

func New(producerID uuid.UUID, name string, altitude int) (*Farm, error) {
	name = helper.ApplyTrim(name)

	if helper.IsDefaultUUID(producerID) {
		return Default(), ErrInvalidProducer
	}
	if !helper.LengthIsValid(name, nameMinLength, nameMaxLength) {
		return Default(), ErrInvalidName
	}
	if !helper.IsInRange(altitude, minAltitude, maxAltitude) {
		return Default(), ErrInvalidAltitude
	}

	return &Farm{
		id:         uuid.New(),
		producerID: producerID,
		name:       name,
		altitude:   altitude,
		createdAt:  time.Now().UTC(),
	}, nil
}

func Restore(id, producerID uuid.UUID, name string, altitude int, createdAt time.Time) *Farm {
	return &Farm{
		id:         id,
		producerID: producerID,
		name:       name,
		altitude:   altitude,
		createdAt:  createdAt,
	}
}

func (f *Farm) BelongsTo(actorID uuid.UUID) bool {
	return !helper.IsDefaultUUID(actorID) && f.producerID == actorID
}

func (f *Farm) ID() uuid.UUID         { return f.id }
func (f *Farm) ProducerID() uuid.UUID { return f.producerID }
func (f *Farm) Name() string          { return f.name }
func (f *Farm) Altitude() int         { return f.altitude }
func (f *Farm) CreatedAt() time.Time  { return f.createdAt }

func (f *Farm) IsDefault() bool {
	return f == nil || helper.IsDefaultUUID(f.id)
}
