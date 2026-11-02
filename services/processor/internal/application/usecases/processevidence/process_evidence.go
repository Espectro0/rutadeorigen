package processevidence

import (
	"context"
	"time"

	"github.com/google/uuid"

	"rutadeorigen/lib/requestcontext"
	"rutadeorigen/processor/internal/application/output/ports"
	"rutadeorigen/processor/internal/domain/evidencefile"
	"rutadeorigen/processor/internal/domain/picture"
	"rutadeorigen/processor/internal/domain/rejection"
	"rutadeorigen/processor/internal/platform/exception"
	"rutadeorigen/processor/internal/platform/helper"
)

const firstEventVersion = 1

var _ UseCase = (*ProcessEvidence)(nil)

type ProcessEvidence struct {
	processed  ports.ProcessedEventRepository
	quarantine ports.QuarantineStorage
	evidences  ports.EvidenceStorage
	images     ports.ImageProcessor
	messages   ports.RejectionMessages
	publisher  ports.EventPublisher
	tx         ports.TxManager
	limits     picture.Limits
}

type storedImage struct {
	checksum     string
	finalKey     string
	thumbnailKey string
	contentType  string
}

func New(processed ports.ProcessedEventRepository, quarantine ports.QuarantineStorage, evidences ports.EvidenceStorage,
	images ports.ImageProcessor, messages ports.RejectionMessages, publisher ports.EventPublisher,
	tx ports.TxManager, limits picture.Limits) *ProcessEvidence {
	return &ProcessEvidence{
		processed:  processed,
		quarantine: quarantine,
		evidences:  evidences,
		images:     images,
		messages:   messages,
		publisher:  publisher,
		tx:         tx,
		limits:     limits,
	}
}

func (uc *ProcessEvidence) Execute(ctx context.Context, cmd Command) error {
	alreadyHandled, err := uc.processed.Exists(ctx, cmd.MessageID)
	if err != nil {
		return exception.Wrap(err, "check processed event")
	}
	if alreadyHandled {
		return nil
	}

	file, err := evidencefile.New(evidencefile.NewParams{
		EvidenceID:    cmd.EvidenceID,
		LotID:         cmd.LotID,
		QuarantineKey: cmd.QuarantineKey,
		SizeBytes:     cmd.SizeBytes,
	})
	if err != nil {
		return err
	}

	stored, err := uc.storeProcessedImage(ctx, file)
	if err != nil {
		reason := rejection.FromError(err)
		if reason.IsDefault() {
			return err
		}
		return uc.reject(ctx, cmd, file, reason)
	}
	return uc.accept(ctx, cmd, file, stored)
}

func (uc *ProcessEvidence) storeProcessedImage(ctx context.Context, file *evidencefile.File) (storedImage, error) {
	failed := storedImage{}

	if err := uc.limits.CheckFileSize(file.SizeBytes()); err != nil {
		return failed, err
	}

	original, err := uc.readOriginal(ctx, file)
	if err != nil {
		return failed, err
	}

	info, err := uc.images.Inspect(ctx, original)
	if err != nil {
		return failed, err
	}
	if err := uc.limits.CheckDimensions(info.Dimensions()); err != nil {
		return failed, err
	}

	rendered, err := uc.images.Process(ctx, original, info, uc.limits.ThumbnailSide())
	if err != nil {
		return failed, err
	}

	finalKey := file.FinalKey(info.Format())
	if err := uc.evidences.Put(ctx, finalKey, info.Format().ContentType(), rendered.Final); err != nil {
		return failed, exception.Wrap(err, "put final image")
	}
	thumbnailKey := file.ThumbnailKey()
	if err := uc.evidences.Put(ctx, thumbnailKey, picture.FormatWebP.ContentType(), rendered.Thumbnail); err != nil {
		return failed, exception.Wrap(err, "put thumbnail")
	}

	return storedImage{
		checksum:     helper.SHA256Hex(original),
		finalKey:     finalKey,
		thumbnailKey: thumbnailKey,
		contentType:  info.Format().ContentType(),
	}, nil
}

func (uc *ProcessEvidence) readOriginal(ctx context.Context, file *evidencefile.File) ([]byte, error) {
	object, err := uc.quarantine.Get(ctx, file.QuarantineKey(), file.MaxDownloadBytes())
	if err != nil {
		return nil, exception.Wrap(err, "get quarantine object")
	}
	if !object.Exists {
		return nil, exception.New(exception.CodeFileMissing)
	}
	if err := file.CheckUploadedSize(int64(len(object.Data))); err != nil {
		return nil, err
	}
	return object.Data, nil
}

func (uc *ProcessEvidence) accept(ctx context.Context, cmd Command, file *evidencefile.File, stored storedImage) error {
	processedAt := time.Now().UTC()
	event := uc.newEvent(ctx, cmd, ports.EventEvidenceProcessed, processedAt, EvidenceProcessedPayload{
		EvidenceID:   file.EvidenceID(),
		LotID:        file.LotID(),
		LotCode:      cmd.LotCode,
		Stage:        cmd.Stage,
		FileName:     cmd.FileName,
		SHA256:       stored.checksum,
		FinalKey:     stored.finalKey,
		ThumbnailKey: stored.thumbnailKey,
		ContentType:  stored.contentType,
		ProcessedAt:  processedAt,
		Uploader:     uploaderPayload(cmd.Uploader),
	})
	return uc.recordOutcome(ctx, cmd, file, event)
}

func (uc *ProcessEvidence) reject(ctx context.Context, cmd Command, file *evidencefile.File, reason rejection.Reason) error {
	processedAt := time.Now().UTC()
	event := uc.newEvent(ctx, cmd, ports.EventEvidenceRejected, processedAt, EvidenceRejectedPayload{
		EvidenceID:  file.EvidenceID(),
		LotID:       file.LotID(),
		LotCode:     cmd.LotCode,
		Stage:       cmd.Stage,
		FileName:    cmd.FileName,
		Reason:      uc.messages.Message(reason),
		ProcessedAt: processedAt,
		Uploader:    uploaderPayload(cmd.Uploader),
	})
	return uc.recordOutcome(ctx, cmd, file, event)
}

func uploaderPayload(uploader Uploader) UploaderPayload {
	return UploaderPayload{ActorID: uploader.ActorID, Role: uploader.Role, Name: uploader.Name, Email: uploader.Email}
}

func (uc *ProcessEvidence) recordOutcome(ctx context.Context, cmd Command, file *evidencefile.File, event ports.Event) error {
	if err := uc.tx.WithinTx(ctx, func(ctx context.Context) error {
		return uc.registerAndPublish(ctx, cmd, event)
	}); err != nil {
		return exception.Wrap(err, "record evidence outcome transaction")
	}

	uc.removeQuarantineCopyBestEffort(ctx, file)
	return nil
}

func (uc *ProcessEvidence) registerAndPublish(ctx context.Context, cmd Command, event ports.Event) error {
	firstTime, err := uc.processed.Register(ctx, cmd.MessageID, cmd.EventType)
	if err != nil {
		return exception.Wrap(err, "register processed event")
	}
	if !firstTime {
		return nil
	}

	if err := uc.publisher.Publish(ctx, event); err != nil {
		return exception.Wrap(err, "publish evidence outcome")
	}
	return nil
}

func (uc *ProcessEvidence) removeQuarantineCopyBestEffort(ctx context.Context, file *evidencefile.File) {
	_ = uc.quarantine.Remove(ctx, file.QuarantineKey())
}

func (uc *ProcessEvidence) newEvent(ctx context.Context, cmd Command, eventType ports.EventType,
	occurredAt time.Time, payload any) ports.Event {
	return ports.Event{
		ID:            uuid.New(),
		Type:          eventType,
		AggregateType: ports.AggregateEvidence,
		AggregateID:   cmd.EvidenceID,
		Version:       firstEventVersion,
		OccurredAt:    occurredAt,
		ActorID:       cmd.Uploader.ActorID,
		ActorRole:     cmd.Uploader.Role,
		CorrelationID: requestcontext.CorrelationID(ctx),
		Payload:       payload,
	}
}
