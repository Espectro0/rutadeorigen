package ports

import (
	"context"

	"github.com/google/uuid"

	"rutadeorigen/traceability/internal/domain/actor"
)

type ActorRepository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*actor.Actor, error)
	FindByUserID(ctx context.Context, userId string) (*actor.Actor, error)
	Save(ctx context.Context, actor *actor.Actor) error
}
