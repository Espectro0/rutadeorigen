package ports

import (
	"context"

	"github.com/google/uuid"

	"rutadeorigen/traceability/internal/domain/actor"
)

type ActorRepository interface {
	FindById(ctx context.Context, id uuid.UUID) (*actor.Actor, error)
}
