package registerstage

import (
	"context"
	"time"

	"rutadeorigen/lib/requestcontext"
	"rutadeorigen/traceability/internal/application/output/ports"
	"rutadeorigen/traceability/internal/application/usecases/access"
	"rutadeorigen/traceability/internal/application/usecases/envelope"
	"rutadeorigen/traceability/internal/application/usecases/validators"
	"rutadeorigen/traceability/internal/domain/actor"
	"rutadeorigen/traceability/internal/domain/lot"
	"rutadeorigen/traceability/internal/platform/exception"
)

var _ UseCase = (*RegisterStage)(nil)

type RegisterStage struct {
	actors    ports.ActorRepository
	lots      ports.LotRepository
	events    ports.LotEventRepository
	auth      ports.Authorizer
	publisher ports.EventPublisher
	tx        ports.TxManager
}

func New(actors ports.ActorRepository, lots ports.LotRepository, events ports.LotEventRepository, auth ports.Authorizer, publisher ports.EventPublisher, tx ports.TxManager) *RegisterStage {
	return &RegisterStage{
		actors:    actors,
		lots:      lots,
		events:    events,
		auth:      auth,
		publisher: publisher,
		tx:        tx,
	}
}

func (uc *RegisterStage) Execute(ctx context.Context, cmd Command) (Result, error) {
	failed := Result{Event: lot.DefaultEvent()}

	a, err := validators.ActorActive(ctx, uc.actors, requestcontext.ActorID(ctx))
	if err != nil {
		return failed, err
	}

	l, err := validators.LotExists(ctx, uc.lots, cmd.LotID)
	if err != nil {
		return failed, err
	}

	resource := ports.Resource{Type: ports.ResourceLot, ID: l.ID(), CustodianID: l.CustodianID()}
	if err := access.Require(ctx, uc.auth, uc.publisher, a, ports.ActionRegisterStage, resource); err != nil {
		return failed, err
	}

	if err := validators.LotVersion(l, cmd.ExpectedVersion); err != nil {
		return failed, err
	}

	var result Result
	err = uc.tx.WithinTx(ctx, func(ctx context.Context) error {
		registered, err := uc.register(ctx, a, l, cmd)
		result = registered
		return err
	})
	if err != nil {
		return failed, exception.Wrap(err, "register stage transaction")
	}

	return result, nil
}

func (uc *RegisterStage) register(ctx context.Context, a *actor.Actor, l *lot.Lot, cmd Command) (Result, error) {
	last, err := uc.events.FindLast(ctx, l.ID())
	if err != nil {
		return Result{}, exception.Wrap(err, "find last lot event")
	}

	if err := l.AdvanceTo(cmd.Stage); err != nil {
		return Result{}, err
	}

	event, err := lot.NewStageEvent(lot.StageEventParams{
		LotID:       l.ID(),
		HarvestDate: l.HarvestDate(),
		Previous:    last,
		Stage:       cmd.Stage,
		EventDate:   cmd.EventDate,
		Note:        cmd.Note,
		ActorID:     a.ID(),
		ActorRole:   a.Type().String(),
		Details:     cmd.Details,
	})
	if err != nil {
		return Result{}, err
	}

	if err := uc.lots.UpdateStage(ctx, l, cmd.ExpectedVersion); err != nil {
		return Result{}, exception.Wrap(err, "update lot stage")
	}
	if err := uc.events.Save(ctx, event); err != nil {
		return Result{}, exception.Wrap(err, "save lot event")
	}
	if err := uc.publishRegistered(ctx, a, l, event); err != nil {
		return Result{}, err
	}

	return Result{Event: event, LotVersion: l.Version()}, nil
}

func (uc *RegisterStage) publishRegistered(ctx context.Context, a *actor.Actor, l *lot.Lot, e *lot.Event) error {
	event := envelope.New(ctx, a, ports.EventLotStageRegistered, ports.AggregateLot, l.ID(), l.Version(), StageRegisteredPayload{
		LotID:     l.ID(),
		Stage:     l.CurrentStage().String(),
		Status:    l.Status().String(),
		EventID:   e.ID(),
		Sequence:  e.Sequence(),
		EventDate: e.EventDate().Format(time.DateOnly),
	})

	if err := uc.publisher.Publish(ctx, event); err != nil {
		return exception.Wrap(err, "publish stage registered")
	}
	return nil
}
