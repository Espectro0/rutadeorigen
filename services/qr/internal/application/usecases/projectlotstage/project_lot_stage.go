package projectlotstage

import (
	"context"

	"rutadeorigen/qr/internal/application/output/ports"
	"rutadeorigen/qr/internal/platform/exception"
	"rutadeorigen/qr/internal/platform/helper"
)

var _ UseCase = (*ProjectLotStage)(nil)

type ProjectLotStage struct {
	views     ports.LotViewRepository
	processed ports.ProcessedEventRepository
	cache     ports.LotViewCache
	tx        ports.TxManager
}

func New(views ports.LotViewRepository, processed ports.ProcessedEventRepository, cache ports.LotViewCache, tx ports.TxManager) *ProjectLotStage {
	return &ProjectLotStage{
		views:     views,
		processed: processed,
		cache:     cache,
		tx:        tx,
	}
}

func (uc *ProjectLotStage) Execute(ctx context.Context, cmd Command) error {
	if helper.IsDefaultUUID(cmd.MessageID) || helper.IsEmpty(cmd.EventType) || helper.IsDefaultUUID(cmd.LotID) {
		return exception.New(exception.CodeEventInvalidPayload)
	}

	err := uc.tx.WithinTx(ctx, func(ctx context.Context) error {
		return uc.project(ctx, cmd)
	})
	if err != nil {
		return exception.Wrap(err, "project lot stage transaction")
	}

	_ = uc.cache.Delete(ctx, cmd.LotID)
	return nil
}

func (uc *ProjectLotStage) project(ctx context.Context, cmd Command) error {
	first, err := uc.processed.Register(ctx, cmd.MessageID, cmd.EventType)
	if err != nil {
		return exception.Wrap(err, "register processed event")
	}
	if !first {
		return nil
	}

	current, err := uc.views.FindByID(ctx, cmd.LotID)
	if err != nil {
		return exception.Wrap(err, "find current lot view")
	}
	if current.IsDefault() {
		return exception.New(exception.CodeLotViewNotFound)
	}
	if !current.IsActive() || cmd.Version <= current.Version() {
		return nil
	}

	updated, err := current.ApplyStage(cmd.Stage, cmd.Version, cmd.OccurredAt)
	if err != nil {
		return err
	}

	if err := uc.views.Save(ctx, updated); err != nil {
		return exception.Wrap(err, "save lot view")
	}
	return nil
}
