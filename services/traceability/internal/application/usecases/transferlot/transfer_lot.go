package transferlot

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

var _ UseCase = (*TransferLot)(nil)

type TransferLot struct {
	actors    ports.ActorRepository
	lots      ports.LotRepository
	events    ports.LotEventRepository
	auth      ports.Authorizer
	publisher ports.EventPublisher
	tx        ports.TxManager
}

func New(actors ports.ActorRepository, lots ports.LotRepository, events ports.LotEventRepository, auth ports.Authorizer, publisher ports.EventPublisher, tx ports.TxManager) *TransferLot {
	return &TransferLot{
		actors:    actors,
		lots:      lots,
		events:    events,
		auth:      auth,
		publisher: publisher,
		tx:        tx,
	}
}

func (uc *TransferLot) Execute(ctx context.Context, cmd Command) (Result, error) {
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
	if err := access.Require(ctx, uc.auth, uc.publisher, a, ports.ActionTransferLot, resource); err != nil {
		return failed, err
	}

	if err := validators.LotVersion(l, cmd.ExpectedVersion); err != nil {
		return failed, err
	}

	target, err := validators.ActorActive(ctx, uc.actors, cmd.TargetActorID)
	if err != nil {
		return failed, err
	}

	var result Result
	err = uc.tx.WithinTx(ctx, func(ctx context.Context) error {
		transferred, err := uc.transfer(ctx, a, target, l, cmd)
		result = transferred
		return err
	})
	if err != nil {
		return failed, exception.Wrap(err, "transfer lot transaction")
	}

	return result, nil
}

func (uc *TransferLot) transfer(ctx context.Context, a, target *actor.Actor, l *lot.Lot, cmd Command) (Result, error) {
	last, err := uc.events.FindLast(ctx, l.ID())
	if err != nil {
		return Result{}, exception.Wrap(err, "find last lot event")
	}

	if err := l.TransferTo(target.ID(), target.Type()); err != nil {
		return Result{}, err
	}

	event, err := lot.NewTransferEvent(lot.TransferEventParams{
		LotID:         l.ID(),
		Previous:      last,
		Stage:         l.CurrentStage(),
		EventDate:     time.Now().UTC(),
		Note:          cmd.Note,
		ActorID:       a.ID(),
		ActorRole:     a.Type().String(),
		TargetActorID: target.ID(),
	})
	if err != nil {
		return Result{}, err
	}

	if err := uc.lots.UpdateCustodian(ctx, l, cmd.ExpectedVersion); err != nil {
		return Result{}, exception.Wrap(err, "update lot custodian")
	}
	if err := uc.events.Save(ctx, event); err != nil {
		return Result{}, exception.Wrap(err, "save lot event")
	}
	if err := uc.publishTransferred(ctx, a, l, event); err != nil {
		return Result{}, err
	}

	return Result{Event: event, LotVersion: l.Version()}, nil
}

func (uc *TransferLot) publishTransferred(ctx context.Context, a *actor.Actor, l *lot.Lot, e *lot.Event) error {
	event := envelope.New(ctx, a, ports.EventLotTransferred, ports.AggregateLot, l.ID(), l.Version(), LotTransferredPayload{
		LotID:       l.ID(),
		FromActorID: a.ID(),
		CustodianID: l.CustodianID(),
		Stage:       l.CurrentStage().String(),
		Status:      l.Status().String(),
		EventID:     e.ID(),
		Sequence:    e.Sequence(),
		EventDate:   e.EventDate().Format(time.DateOnly),
	})

	if err := uc.publisher.Publish(ctx, event); err != nil {
		return exception.Wrap(err, "publish lot transferred")
	}
	return nil
}
