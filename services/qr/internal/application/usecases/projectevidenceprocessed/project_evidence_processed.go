package projectevidenceprocessed

import (
	"context"

	"rutadeorigen/qr/internal/application/output/ports"
	"rutadeorigen/qr/internal/domain/evidenceview"
	"rutadeorigen/qr/internal/domain/lotview"
	"rutadeorigen/qr/internal/platform/exception"
	"rutadeorigen/qr/internal/platform/helper"
)

var _ UseCase = (*ProjectEvidenceProcessed)(nil)

type ProjectEvidenceProcessed struct {
	evidences ports.EvidenceViewRepository
	processed ports.ProcessedEventRepository
	tx        ports.TxManager
}

func New(evidences ports.EvidenceViewRepository, processed ports.ProcessedEventRepository, tx ports.TxManager) *ProjectEvidenceProcessed {
	return &ProjectEvidenceProcessed{evidences: evidences, processed: processed, tx: tx}
}

func (uc *ProjectEvidenceProcessed) Execute(ctx context.Context, cmd Command) error {
	if helper.IsDefaultUUID(cmd.MessageID) || helper.IsEmpty(cmd.EventType) {
		return exception.New(exception.CodeEventInvalidPayload)
	}

	view, err := evidenceview.New(evidenceview.NewParams{
		EvidenceID:   cmd.EvidenceID,
		LotID:        cmd.LotID,
		Stage:        lotview.ParseStage(cmd.Stage),
		FinalKey:     cmd.FinalKey,
		ThumbnailKey: cmd.ThumbnailKey,
		ContentType:  cmd.ContentType,
		ProcessedAt:  cmd.ProcessedAt,
	})
	if err != nil {
		return err
	}

	err = uc.tx.WithinTx(ctx, func(ctx context.Context) error {
		return uc.project(ctx, cmd, view)
	})
	if err != nil {
		return exception.Wrap(err, "project evidence processed transaction")
	}
	return nil
}

func (uc *ProjectEvidenceProcessed) project(ctx context.Context, cmd Command, view *evidenceview.EvidenceView) error {
	firstTime, err := uc.processed.Register(ctx, cmd.MessageID, cmd.EventType)
	if err != nil {
		return exception.Wrap(err, "register processed event")
	}
	if !firstTime {
		return nil
	}

	if err := uc.evidences.Save(ctx, view); err != nil {
		return exception.Wrap(err, "save evidence view")
	}
	return nil
}
