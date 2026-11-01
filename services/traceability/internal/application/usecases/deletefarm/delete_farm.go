package deletefarm

import (
	"context"

	"github.com/google/uuid"

	"rutadeorigen/lib/requestcontext"
	"rutadeorigen/traceability/internal/application/output/ports"
	"rutadeorigen/traceability/internal/application/usecases/access"
	"rutadeorigen/traceability/internal/application/usecases/validators"
	"rutadeorigen/traceability/internal/domain/lot"
	"rutadeorigen/traceability/internal/platform/exception"
)

var _ UseCase = (*DeleteFarm)(nil)

type DeleteFarm struct {
	actors    ports.ActorRepository
	farms     ports.FarmRepository
	lots      ports.LotRepository
	auth      ports.Authorizer
	publisher ports.EventPublisher
}

func New(actors ports.ActorRepository, farms ports.FarmRepository, lots ports.LotRepository, auth ports.Authorizer, publisher ports.EventPublisher) *DeleteFarm {
	return &DeleteFarm{actors: actors, farms: farms, lots: lots, auth: auth, publisher: publisher}
}

func (uc *DeleteFarm) Execute(ctx context.Context, id uuid.UUID) error {
	a, err := validators.ActorActive(ctx, uc.actors, requestcontext.ActorID(ctx))
	if err != nil {
		return err
	}

	f, err := validators.FarmExists(ctx, uc.farms, id)
	if err != nil {
		return err
	}

	resource := ports.Resource{Type: ports.ResourceFarm, ID: f.ID(), OwnerID: f.ProducerID()}
	if err := access.Require(ctx, uc.auth, uc.publisher, a, ports.ActionDeleteFarm, resource); err != nil {
		return err
	}

	lots, err := uc.lots.List(ctx, lot.Filter{FarmID: f.ID()}, lot.Cursor{}, 1)
	if err != nil {
		return exception.Wrap(err, "list farm lots")
	}
	if len(lots) > 0 {
		return exception.New(exception.CodeFarmHasLots)
	}

	if err := uc.farms.Delete(ctx, f.ID()); err != nil {
		return exception.Wrap(err, "delete farm")
	}
	return nil
}
