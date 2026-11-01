package voidlot

import (
	"context"

	"rutadeorigen/lib/requestcontext"
	"rutadeorigen/traceability/internal/application/output/ports"
	"rutadeorigen/traceability/internal/application/usecases/access"
	"rutadeorigen/traceability/internal/application/usecases/envelope"
	"rutadeorigen/traceability/internal/application/usecases/validators"
	"rutadeorigen/traceability/internal/domain/actor"
	"rutadeorigen/traceability/internal/domain/lot"
	"rutadeorigen/traceability/internal/platform/exception"
)

var _ UseCase = (*VoidLot)(nil)

type VoidLot struct {
	actors    ports.ActorRepository
	lots      ports.LotRepository
	auth      ports.Authorizer
	publisher ports.EventPublisher
	tx        ports.TxManager
}

func New(actors ports.ActorRepository, lots ports.LotRepository, auth ports.Authorizer, publisher ports.EventPublisher, tx ports.TxManager) *VoidLot {
	return &VoidLot{actors: actors, lots: lots, auth: auth, publisher: publisher, tx: tx}
}

func (uc *VoidLot) Execute(ctx context.Context, cmd Command) (*lot.Lot, error) {
	a, err := validators.ActorActive(ctx, uc.actors, requestcontext.ActorID(ctx))
	if err != nil {
		return lot.Default(), err
	}

	l, err := validators.LotExists(ctx, uc.lots, cmd.LotID)
	if err != nil {
		return lot.Default(), err
	}

	resource := ports.Resource{Type: ports.ResourceLot, ID: l.ID(), CustodianID: l.CustodianID()}
	if err := access.Require(ctx, uc.auth, uc.publisher, a, ports.ActionVoidLot, resource); err != nil {
		return lot.Default(), err
	}

	if err := validators.LotVersion(l, cmd.ExpectedVersion); err != nil {
		return lot.Default(), err
	}

	err = uc.tx.WithinTx(ctx, func(ctx context.Context) error {
		return uc.void(ctx, a, l, cmd)
	})
	if err != nil {
		return lot.Default(), exception.Wrap(err, "void lot transaction")
	}

	return l, nil
}

func (uc *VoidLot) void(ctx context.Context, a *actor.Actor, l *lot.Lot, cmd Command) error {
	if err := l.Void(cmd.Reason); err != nil {
		return err
	}

	if err := uc.lots.UpdateStatus(ctx, l, cmd.ExpectedVersion); err != nil {
		return exception.Wrap(err, "update lot status")
	}

	event := envelope.New(ctx, a, ports.EventLotVoided, ports.AggregateLot, l.ID(), l.Version(), LotVoidedPayload{
		LotID:       l.ID(),
		VoidedBy:    a.ID(),
		Reason:      l.VoidReason(),
		CustodianID: l.CustodianID(),
		Stage:       l.CurrentStage().String(),
		Status:      l.Status().String(),
	})
	if err := uc.publisher.Publish(ctx, event); err != nil {
		return exception.Wrap(err, "publish lot voided")
	}
	return nil
}
