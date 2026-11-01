package projectactorregistered

import (
	"context"

	"rutadeorigen/qr/internal/application/output/ports"
	"rutadeorigen/qr/internal/domain/actorview"
	"rutadeorigen/qr/internal/platform/exception"
	"rutadeorigen/qr/internal/platform/helper"
)

var _ UseCase = (*ProjectActorRegistered)(nil)

type ProjectActorRegistered struct {
	actors    ports.ActorViewRepository
	processed ports.ProcessedEventRepository
	tx        ports.TxManager
}

func New(actors ports.ActorViewRepository, processed ports.ProcessedEventRepository, tx ports.TxManager) *ProjectActorRegistered {
	return &ProjectActorRegistered{actors: actors, processed: processed, tx: tx}
}

func (uc *ProjectActorRegistered) Execute(ctx context.Context, cmd Command) error {
	if helper.IsDefaultUUID(cmd.MessageID) || helper.IsEmpty(cmd.EventType) {
		return exception.New(exception.CodeEventInvalidPayload)
	}

	err := uc.tx.WithinTx(ctx, func(ctx context.Context) error {
		return uc.project(ctx, cmd)
	})
	if err != nil {
		return exception.Wrap(err, "project actor registered transaction")
	}
	return nil
}

func (uc *ProjectActorRegistered) project(ctx context.Context, cmd Command) error {
	first, err := uc.processed.Register(ctx, cmd.MessageID, cmd.EventType)
	if err != nil {
		return exception.Wrap(err, "register processed event")
	}
	if !first {
		return nil
	}

	view, err := actorview.New(actorview.NewParams{
		ActorID:   cmd.ActorID,
		UserID:    cmd.UserID,
		Role:      cmd.Role,
		CreatedAt: cmd.OccurredAt,
	})
	if err != nil {
		return err
	}

	if err := uc.actors.Save(ctx, view); err != nil {
		return exception.Wrap(err, "save actor view")
	}
	return nil
}
