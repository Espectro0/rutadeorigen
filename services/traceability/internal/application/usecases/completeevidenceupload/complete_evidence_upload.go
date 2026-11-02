package completeevidenceupload

import (
	"context"

	"rutadeorigen/lib/requestcontext"
	"rutadeorigen/traceability/internal/application/output/ports"
	"rutadeorigen/traceability/internal/application/usecases/envelope"
	"rutadeorigen/traceability/internal/application/usecases/validators"
	"rutadeorigen/traceability/internal/domain/actor"
	"rutadeorigen/traceability/internal/domain/evidence"
	"rutadeorigen/traceability/internal/domain/lot"
	"rutadeorigen/traceability/internal/platform/exception"
)

var _ UseCase = (*CompleteEvidenceUpload)(nil)

type CompleteEvidenceUpload struct {
	actors    ports.ActorRepository
	lots      ports.LotRepository
	evidences ports.EvidenceRepository
	storage   ports.EvidenceStorage
	publisher ports.EventPublisher
	tx        ports.TxManager
}

func New(actors ports.ActorRepository, lots ports.LotRepository, evidences ports.EvidenceRepository,
	storage ports.EvidenceStorage, publisher ports.EventPublisher, tx ports.TxManager) *CompleteEvidenceUpload {
	return &CompleteEvidenceUpload{
		actors:    actors,
		lots:      lots,
		evidences: evidences,
		storage:   storage,
		publisher: publisher,
		tx:        tx,
	}
}

func (uc *CompleteEvidenceUpload) Execute(ctx context.Context, cmd Command) (*evidence.Evidence, error) {
	failed := evidence.Default()

	a, err := validators.ActorActive(ctx, uc.actors, requestcontext.ActorID(ctx))
	if err != nil {
		return failed, err
	}

	e, err := uc.evidences.FindByID(ctx, cmd.EvidenceID)
	if err != nil {
		return failed, exception.Wrap(err, "find evidence by id")
	}
	if e.IsDefault() || e.LotID() != cmd.LotID {
		return failed, exception.New(exception.CodeEvidenceNotFound)
	}
	if !e.IsUploadedBy(a.ID()) {
		return failed, exception.New(exception.CodeForbidden)
	}

	// Completing twice is harmless: the event was already published.
	if e.Status() != evidence.StatusPendingUpload {
		return e, nil
	}

	l, err := validators.LotExists(ctx, uc.lots, e.LotID())
	if err != nil {
		return failed, err
	}

	if err := uc.verifyQuarantine(ctx, e); err != nil {
		return failed, err
	}

	if err := uc.tx.WithinTx(ctx, func(ctx context.Context) error {
		return uc.confirm(ctx, a, l, e)
	}); err != nil {
		return failed, exception.Wrap(err, "complete evidence upload transaction")
	}

	return e, nil
}

func (uc *CompleteEvidenceUpload) verifyQuarantine(ctx context.Context, e *evidence.Evidence) error {
	info, err := uc.storage.Stat(ctx, e.QuarantineKey())
	if err != nil {
		return exception.Wrap(err, "stat quarantine object")
	}
	if !info.Exists {
		return exception.New(exception.CodeEvidenceFileMissing)
	}
	if info.SizeBytes != e.SizeBytes() {
		// The file is not what was announced: it must not reach the processor.
		if err := uc.storage.Remove(ctx, e.QuarantineKey()); err != nil {
			return exception.Wrap(err, "remove mismatched quarantine object")
		}
		return exception.New(exception.CodeEvidenceFileMismatch)
	}
	return nil
}

func (uc *CompleteEvidenceUpload) confirm(ctx context.Context, a *actor.Actor, l *lot.Lot, e *evidence.Evidence) error {
	if err := e.MarkUploaded(); err != nil {
		return err
	}
	if err := uc.evidences.Update(ctx, e); err != nil {
		return exception.Wrap(err, "update evidence")
	}

	event := envelope.New(ctx, a, ports.EventEvidenceUploaded, ports.AggregateEvidence, e.ID(), l.Version(), EvidenceUploadedPayload{
		EvidenceID:    e.ID(),
		LotID:         l.ID(),
		LotCode:       l.Code(),
		Stage:         e.Stage().String(),
		FileName:      e.FileName(),
		ContentType:   e.ContentType().String(),
		SizeBytes:     e.SizeBytes(),
		QuarantineKey: e.QuarantineKey(),
		Uploader: Uploader{
			ActorID: a.ID(),
			Role:    a.Type().String(),
			Name:    a.Name(),
			Email:   a.Email(),
		},
	})
	if err := uc.publisher.Publish(ctx, event); err != nil {
		return exception.Wrap(err, "publish evidence uploaded")
	}
	return nil
}
