package createlot

import (
	"context"
	"time"

	"github.com/google/uuid"

	"rutadeorigen/lib/requestcontext"
	"rutadeorigen/traceability/internal/application/output/ports"
	"rutadeorigen/traceability/internal/application/usecases/access"
	"rutadeorigen/traceability/internal/application/usecases/envelope"
	"rutadeorigen/traceability/internal/application/usecases/validators"
	"rutadeorigen/traceability/internal/domain/actor"
	"rutadeorigen/traceability/internal/domain/farm"
	"rutadeorigen/traceability/internal/domain/lot"
	"rutadeorigen/traceability/internal/platform/exception"
	"rutadeorigen/traceability/internal/platform/helper"
)

var _ UseCase = (*CreateLot)(nil)

type CreateLot struct {
	actors    ports.ActorRepository
	farms     ports.FarmRepository
	lots      ports.LotRepository
	auth      ports.Authorizer
	publisher ports.EventPublisher
	tx        ports.TxManager
}

func New(actors ports.ActorRepository, farms ports.FarmRepository, lots ports.LotRepository, auth ports.Authorizer, publisher ports.EventPublisher, tx ports.TxManager) *CreateLot {
	return &CreateLot{
		actors:    actors,
		farms:     farms,
		lots:      lots,
		auth:      auth,
		publisher: publisher,
		tx:        tx,
	}
}

func (uc *CreateLot) Execute(ctx context.Context, draft *lot.Lot) error {
	if helper.IsNil(draft) {
		draft = lot.Default()
	}

	a, err := validators.ActorActive(ctx, uc.actors, requestcontext.ActorID(ctx))
	if err != nil {
		return err
	}

	f, err := validators.FarmExists(ctx, uc.farms, draft.FarmID())
	if err != nil {
		return err
	}

	resource := ports.Resource{Type: ports.ResourceFarm, ID: f.ID(), OwnerID: f.ProducerID()}
	if err := access.Require(ctx, uc.auth, uc.publisher, a, ports.ActionCreateLot, resource); err != nil {
		return err
	}

	created, err := lot.New(draft.FarmID(), f.ProducerID(), draft.Code(),
		draft.Variety(), draft.Process(), draft.HarvestDate())
	if err != nil {
		return err
	}

	if err := uc.ensureCodeAvailable(ctx, created.FarmID(), created.Code()); err != nil {
		return err
	}

	err = uc.tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := uc.lots.Save(ctx, created); err != nil {
			return exception.Wrap(err, "save lot")
		}
		return uc.publishCreated(ctx, f, a, created)
	})
	if err != nil {
		return exception.Wrap(err, "create lot transaction")
	}

	*draft = *created
	return nil
}

func (uc *CreateLot) ensureCodeAvailable(ctx context.Context, farmID uuid.UUID, code string) error {
	exists, err := uc.lots.ExistsByCode(ctx, farmID, helper.ApplyTrim(code))
	if err != nil {
		return exception.Wrap(err, "check lot code")
	}
	if exists {
		return exception.New(exception.CodeLotCodeAlreadyExists)
	}
	return nil
}

func (uc *CreateLot) publishCreated(ctx context.Context, f *farm.Farm, a *actor.Actor, l *lot.Lot) error {
	producer, err := validators.ActorActive(ctx, uc.actors, f.ProducerID())
	if err != nil {
		return exception.Wrap(err, "get producer")
	}

	event := envelope.New(ctx, a, ports.EventLotCreated, ports.AggregateLot, l.ID(), l.Version(), LotCreatedPayload{
		LotID: l.ID(),
		Farm: FarmSnapshot{
			ID:       f.ID(),
			Name:     f.Name(),
			Altitude: f.Altitude(),
		},
		Producer: ProducerSnapshot{
			ID:   producer.ID(),
			Name: producer.Name(),
		},
		CustodianID: l.CustodianID(),
		Code:        l.Code(),
		Variety:     l.Variety(),
		Process:     l.Process().String(),
		HarvestDate: l.HarvestDate().Format(time.DateOnly),
		Stage:       l.CurrentStage().String(),
		Status:      l.Status().String(),
	})

	if err := uc.publisher.Publish(ctx, event); err != nil {
		return exception.Wrap(err, "publish lot created")
	}
	return nil
}
