package ports

import (
	"context"

	"github.com/google/uuid"

	"rutadeorigen/traceability/internal/domain/actor"
)

type ActorRepository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*actor.Actor, error)
}
