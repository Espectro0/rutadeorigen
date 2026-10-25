package envelope

import (
	"context"
	"time"

	"github.com/google/uuid"

	"rutadeorigen/lib/requestcontext"
	"rutadeorigen/traceability/internal/application/output/ports"
	"rutadeorigen/traceability/internal/domain/actor"
)

func New(ctx context.Context, a *actor.Actor, eventType ports.EventType, aggregateType ports.AggregateType,
	aggregateID uuid.UUID, version int, payload any) ports.Event {
	return ports.Event{
		ID:            uuid.New(),
		Type:          eventType,
		AggregateType: aggregateType,
		AggregateID:   aggregateID,
		Version:       version,
		OccurredAt:    time.Now().UTC(),
		ActorID:       a.ID(),
		ActorRole:     a.Type().String(),
		CorrelationID: requestcontext.CorrelationID(ctx),
		Payload:       payload,
	}
}
