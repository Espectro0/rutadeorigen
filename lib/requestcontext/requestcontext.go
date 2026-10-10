package requestcontext

import (
	"context"

	"github.com/google/uuid"
)

type key int

const (
	actorIDKey key = iota
	correlationIDKey
)

func WithActorID(ctx context.Context, id uuid.UUID) context.Context {
	return context.WithValue(ctx, actorIDKey, id)
}

func ActorID(ctx context.Context) uuid.UUID {
	if id, ok := ctx.Value(actorIDKey).(uuid.UUID); ok {
		return id
	}
	return uuid.Nil
}

func WithCorrelationID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, correlationIDKey, id)
}

func CorrelationID(ctx context.Context) string {
	if id, ok := ctx.Value(correlationIDKey).(string); ok {
		return id
	}
	return ""
}
