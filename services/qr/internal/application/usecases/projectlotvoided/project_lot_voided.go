package projectlotvoided

import (
	"context"

	"rutadeorigen/qr/internal/application/output/ports"
	"rutadeorigen/qr/internal/platform/exception"
	"rutadeorigen/qr/internal/platform/helper"
)

var _ UseCase = (*ProjectLotVoided)(nil)

type ProjectLotVoided struct {
	views     ports.LotViewRepository
	processed ports.ProcessedEventRepository
	cache     ports.LotViewCache
	tx        ports.TxManager
}

func New(views ports.LotViewRepository, processed ports.ProcessedEventRepository, cache ports.LotViewCache, tx ports.TxManager) *ProjectLotVoided {
	return &ProjectLotVoided{views: views, processed: processed, cache: cache, tx: tx}
}

func (uc *ProjectLotVoided) Execute(ctx context.Context, cmd Command) error {
	if helper.IsDefaultUUID(cmd.MessageID) || helper.IsEmpty(cmd.EventType) || helper.IsDefaultUUID(cmd.LotID) {
		return exception.New(exception.CodeEventInvalidPayload)
	}

	err := uc.tx.WithinTx(ctx, func(ctx context.Context) error {
		return uc.project(ctx, cmd)
	})
	if err != nil {
		return exception.Wrap(err, "project lot voided transaction")
	}

	_ = uc.cache.Delete(ctx, cmd.LotID)
	return nil
}

func (uc *ProjectLotVoided) project(ctx context.Context, cmd Command) error {
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

	updated, err := current.ApplyVoid(cmd.Reason, cmd.Version, cmd.OccurredAt)
	if err != nil {
		return err
	}

	if err := uc.views.Save(ctx, updated); err != nil {
		return exception.Wrap(err, "save lot view")
	}
	return nil
}
