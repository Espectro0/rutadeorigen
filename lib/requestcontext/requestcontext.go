package requestcontext

import (
	"context"

	"github.com/google/uuid"
)

type key int

type Identity struct {
	UserID    string
	Email     string
	FirstName *string
	LastName  *string
}

const (
	actorIDKey key = iota
	correlationIDKey
	userIDKey
	emailKey
	firstNameKey
	lastNameKey
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

func WithUserID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, userIDKey, id)
}

func UserID(ctx context.Context) string {
	if id, ok := ctx.Value(userIDKey).(string); ok {
		return id
	}
	return ""
}

func WithEmail(ctx context.Context, email string) context.Context {
	return context.WithValue(ctx, emailKey, email)
}

func Email(ctx context.Context) string {
	if email, ok := ctx.Value(emailKey).(string); ok {
		return email
	}
	return ""
}

func WithFirstName(ctx context.Context, name string) context.Context {
	return context.WithValue(ctx, firstNameKey, name)
}

func FirstName(ctx context.Context) string {
	if name, ok := ctx.Value(firstNameKey).(string); ok {
		return name
	}
	return ""
}

func WithLastName(ctx context.Context, name string) context.Context {
	return context.WithValue(ctx, lastNameKey, name)
}

func LastName(ctx context.Context) string {
	if name, ok := ctx.Value(lastNameKey).(string); ok {
		return name
	}
	return ""
}
