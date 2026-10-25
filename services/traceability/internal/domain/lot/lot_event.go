package lot

import (
	"encoding/json"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"rutadeorigen/traceability/internal/platform/exception"
	"rutadeorigen/traceability/internal/platform/helper"
)

const (
	GenesisHash   = "0000000000000000000000000000000000000000000000000000000000000000"
	noteMaxLength = 500
	firstSequence = 1
)

type Event struct {
	id           uuid.UUID
	lotID        uuid.UUID
	sequence     int
	eventType    EventType
	stage        Stage
	eventDate    time.Time
	note         string
	actorID      uuid.UUID
	actorRole    string
	details      StageDetails
	previousHash string
	hash         string
	registeredAt time.Time
}

var defaultEvent = Event{
	id:           helper.DefaultUUID(),
	lotID:        helper.DefaultUUID(),
	sequence:     helper.DefaultInt(),
	eventType:    EventTypeUnknown,
	stage:        StageUnknown,
	eventDate:    helper.DefaultDate(),
	note:         helper.EmptyString,
	actorID:      helper.DefaultUUID(),
	actorRole:    helper.EmptyString,
	details:      nil,
	previousHash: helper.EmptyString,
	hash:         helper.EmptyString,
	registeredAt: helper.DefaultDate(),
}

func DefaultEvent() *Event {
	e := defaultEvent
	return &e
}

type StageEventParams struct {
	LotID       uuid.UUID
	HarvestDate time.Time
	Previous    *Event
	Stage       Stage
	EventDate   time.Time
	Note        string
	ActorID     uuid.UUID
	ActorRole   string
	Details     StageDetails
}

func NewStageEvent(p StageEventParams) (*Event, error) {
	note := helper.ApplyTrim(p.Note)
	role := helper.ApplyTrim(p.ActorRole)

	if helper.IsDefaultUUID(p.LotID) || helper.IsDefaultUUID(p.ActorID) || helper.IsEmpty(role) {
		return DefaultEvent(), exception.New(exception.CodeInternal).
			WithTechnical("lot event requires a lot, an actor and an actor role")
	}
	if !p.Stage.IsValid() || p.Stage == StageNone {
		return DefaultEvent(), exception.New(exception.CodeLotInvalidStage)
	}
	if utf8.RuneCountInString(note) > noteMaxLength {
		return DefaultEvent(), exception.New(exception.CodeLotInvalidNote).
			WithParams(map[string]any{"max": noteMaxLength})
	}

	eventDate := helper.DateUTC(p.EventDate)
	if helper.IsDefaultDate(p.EventDate) || helper.IsFutureDate(p.EventDate) ||
		eventDate.Before(helper.DateUTC(p.HarvestDate)) {
		return DefaultEvent(), exception.New(exception.CodeLotInvalidEventDate)
	}

	if p.Details == nil || p.Details.Stage() != p.Stage {
		return DefaultEvent(), exception.New(exception.CodeStageInvalidDetails)
	}
	if err := p.Details.Validate(eventDate); err != nil {
		return DefaultEvent(), err
	}

	sequence, previousHash, err := chainPosition(p.LotID, p.Previous)
	if err != nil {
		return DefaultEvent(), err
	}

	e := &Event{
		id:           uuid.New(),
		lotID:        p.LotID,
		sequence:     sequence,
		eventType:    EventTypeStage,
		stage:        p.Stage,
		eventDate:    eventDate,
		note:         note,
		actorID:      p.ActorID,
		actorRole:    role,
		details:      p.Details,
		previousHash: previousHash,
		registeredAt: time.Now().UTC().Truncate(time.Microsecond),
	}

	e.hash = e.ComputeHash()
	if helper.IsEmpty(e.hash) {
		return DefaultEvent(), exception.New(exception.CodeInternal).
			WithTechnical("compute lot event hash")
	}
	return e, nil
}

func chainPosition(lotID uuid.UUID, previous *Event) (int, string, error) {
	if previous.IsDefault() {
		return firstSequence, GenesisHash, nil
	}
	if previous.lotID != lotID {
		return 0, helper.EmptyString, exception.New(exception.CodeInternal).
			WithTechnical("previous lot event belongs to another lot")
	}
	return previous.sequence + 1, previous.hash, nil
}

type RestoreEventParams struct {
	ID           uuid.UUID
	LotID        uuid.UUID
	Sequence     int
	EventType    EventType
	Stage        Stage
	EventDate    time.Time
	Note         string
	ActorID      uuid.UUID
	ActorRole    string
	Details      StageDetails
	PreviousHash string
	Hash         string
	RegisteredAt time.Time
}

func RestoreEvent(p RestoreEventParams) *Event {
	return &Event{
		id:           p.ID,
		lotID:        p.LotID,
		sequence:     p.Sequence,
		eventType:    p.EventType,
		stage:        p.Stage,
		eventDate:    p.EventDate,
		note:         p.Note,
		actorID:      p.ActorID,
		actorRole:    p.ActorRole,
		details:      p.Details,
		previousHash: p.PreviousHash,
		hash:         p.Hash,
		registeredAt: p.RegisteredAt,
	}
}

type hashInput struct {
	PreviousHash string       `json:"previous_hash"`
	LotID        string       `json:"lot_id"`
	Sequence     int          `json:"sequence"`
	Type         string       `json:"type"`
	Stage        string       `json:"stage"`
	EventDate    string       `json:"event_date"`
	Note         string       `json:"note"`
	ActorID      string       `json:"actor_id"`
	ActorRole    string       `json:"actor_role"`
	Details      StageDetails `json:"details"`
	RegisteredAt string       `json:"registered_at"`
}

func (e *Event) ComputeHash() string {
	payload, err := json.Marshal(hashInput{
		PreviousHash: e.previousHash,
		LotID:        e.lotID.String(),
		Sequence:     e.sequence,
		Type:         e.eventType.String(),
		Stage:        e.stage.String(),
		EventDate:    e.eventDate.UTC().Format(time.DateOnly),
		Note:         e.note,
		ActorID:      e.actorID.String(),
		ActorRole:    e.actorRole,
		Details:      e.details,
		RegisteredAt: e.registeredAt.UTC().Format(time.RFC3339Nano),
	})
	if err != nil {
		return helper.EmptyString
	}
	return helper.SHA256Hex(payload)
}

func (e *Event) VerifyHash() bool {
	return !e.IsDefault() && e.hash == e.ComputeHash()
}

func (e *Event) ID() uuid.UUID           { return e.id }
func (e *Event) LotID() uuid.UUID        { return e.lotID }
func (e *Event) Sequence() int           { return e.sequence }
func (e *Event) Type() EventType         { return e.eventType }
func (e *Event) Stage() Stage            { return e.stage }
func (e *Event) EventDate() time.Time    { return e.eventDate }
func (e *Event) Note() string            { return e.note }
func (e *Event) ActorID() uuid.UUID      { return e.actorID }
func (e *Event) ActorRole() string       { return e.actorRole }
func (e *Event) Details() StageDetails   { return e.details }
func (e *Event) PreviousHash() string    { return e.previousHash }
func (e *Event) Hash() string            { return e.hash }
func (e *Event) RegisteredAt() time.Time { return e.registeredAt }

func (e *Event) IsDefault() bool {
	return e == nil || helper.IsDefaultUUID(e.id)
}
