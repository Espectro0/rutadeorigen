package ports

import (
	"context"

	"rutadeorigen/qr/internal/domain/actorview"
)

type ActorViewRepository interface {
	FindByUserID(ctx context.Context, userID string) (*actorview.ActorView, error)
	Save(ctx context.Context, view *actorview.ActorView) error
}
