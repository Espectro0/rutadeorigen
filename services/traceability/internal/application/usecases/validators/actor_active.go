package validators

import (
	"context"

	"github.com/google/uuid"

	"rutadeorigen/traceability/internal/application/output/ports"
	"rutadeorigen/traceability/internal/application/usecases"
	"rutadeorigen/traceability/internal/domain/actor"
	"rutadeorigen/traceability/internal/platform/exception"
)

func ActorActive(ctx context.Context, repo ports.ActorRepository, id uuid.UUID) (*actor.Actor, error) {
	a, err := repo.FindById(ctx, id)
	if err != nil {
		return actor.Default(), exception.Wrap(err, exception.LayerApplication, "find actor by id")
	}
	if a.IsDefault() || !a.IsActive() {
		return actor.Default(), usecases.ErrActorNotFound
	}
	return a, nil
}
