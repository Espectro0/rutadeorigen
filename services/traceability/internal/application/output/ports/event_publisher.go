package ports

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type EventType string

const (
	EventLotCreated   EventType = "lot.created"
	EventAccessDenied EventType = "security.access_denied"
)

type AggregateType string

const (
	AggregateLot AggregateType = "lot"
)

type Event struct {
	ID            uuid.UUID
	Type          EventType
	AggregateType AggregateType
	AggregateID   uuid.UUID
	Version       int
	OccurredAt    time.Time
	ActorID       uuid.UUID
	ActorRole     string
	CorrelationID string
	Payload       any
}

type EventPublisher interface {
	Publish(ctx context.Context, event Event) error
}
