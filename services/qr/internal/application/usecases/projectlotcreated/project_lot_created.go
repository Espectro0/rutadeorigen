package projectlotcreated

import (
	"context"

	"rutadeorigen/qr/internal/application/output/ports"
	"rutadeorigen/qr/internal/domain/lotview"
	"rutadeorigen/qr/internal/platform/exception"
	"rutadeorigen/qr/internal/platform/helper"
)

var _ UseCase = (*ProjectLotCreated)(nil)

type ProjectLotCreated struct {
	views     ports.LotViewRepository
	processed ports.ProcessedEventRepository
	cache     ports.LotViewCache
	tx        ports.TxManager
}

func New(views ports.LotViewRepository, processed ports.ProcessedEventRepository, cache ports.LotViewCache, tx ports.TxManager) *ProjectLotCreated {
	return &ProjectLotCreated{
		views:     views,
		processed: processed,
		cache:     cache,
		tx:        tx,
	}
}

func (uc *ProjectLotCreated) Execute(ctx context.Context, cmd Command) error {
	if helper.IsDefaultUUID(cmd.MessageID) || helper.IsEmpty(cmd.EventType) {
		return exception.New(exception.CodeEventInvalidPayload)
	}

	view, err := lotview.New(cmd.Lot)
	if err != nil {
		return err
	}

	err = uc.tx.WithinTx(ctx, func(ctx context.Context) error {
		return uc.project(ctx, cmd, view)
	})
	if err != nil {
		return exception.Wrap(err, "project lot created transaction")
	}

	_ = uc.cache.Delete(ctx, view.LotID())
	return nil
}

func (uc *ProjectLotCreated) project(ctx context.Context, cmd Command, view *lotview.LotView) error {
	first, err := uc.processed.Register(ctx, cmd.MessageID, cmd.EventType)
	if err != nil {
		return exception.Wrap(err, "register processed event")
	}
	if !first {
		return nil
	}

	current, err := uc.views.FindByID(ctx, view.LotID())
	if err != nil {
		return exception.Wrap(err, "find current lot view")
	}
	if !view.IsNewerThan(current) {
		return nil
	}

	if err := uc.views.Save(ctx, view); err != nil {
		return exception.Wrap(err, "save lot view")
	}
	return nil
}
