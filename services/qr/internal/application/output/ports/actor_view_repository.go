package ports

import (
	"context"

	"github.com/google/uuid"

	"rutadeorigen/qr/internal/domain/actorview"
)

type ActorViewRepository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*actorview.ActorView, error)
	FindByUserID(ctx context.Context, userID string) (*actorview.ActorView, error)
	Save(ctx context.Context, view *actorview.ActorView) error
}
