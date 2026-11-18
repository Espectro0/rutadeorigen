package sendevidencenotification

import (
	"context"
	"time"

	"github.com/google/uuid"

	"rutadeorigen/notifications/internal/application/output/ports"
	"rutadeorigen/notifications/internal/domain/notification"
	"rutadeorigen/notifications/internal/platform/exception"
)

var _ UseCase = (*SendEvidenceNotification)(nil)

type SendEvidenceNotification struct {
	processed ports.ProcessedEventRepository
	log       ports.NotificationLogRepository
	renderer  ports.EmailRenderer
	sender    ports.EmailSender
	tx        ports.TxManager
}

func New(processed ports.ProcessedEventRepository, log ports.NotificationLogRepository,
	renderer ports.EmailRenderer, sender ports.EmailSender, tx ports.TxManager) *SendEvidenceNotification {
	return &SendEvidenceNotification{
		processed: processed,
		log:       log,
		renderer:  renderer,
		sender:    sender,
		tx:        tx,
	}
}

func (uc *SendEvidenceNotification) Execute(ctx context.Context, cmd Command) error {
	alreadyHandled, err := uc.processed.Exists(ctx, cmd.MessageID)
	if err != nil {
		return exception.Wrap(err, "check processed event")
	}
	if alreadyHandled {
		return nil
	}

	candidate, err := uc.newNotification(cmd)
	if err != nil {
		return err
	}

	stored, err := uc.log.Reserve(ctx, candidate)
	if err != nil {
		return exception.Wrap(err, "reserve notification")
	}
	if !stored.IsPending() {
		return uc.markHandled(ctx, cmd)
	}

	return uc.deliver(ctx, cmd, stored)
}

func (uc *SendEvidenceNotification) newNotification(cmd Command) (*notification.Notification, error) {
	recipient, err := notification.NewRecipient(cmd.RecipientName, cmd.RecipientEmail)
	if err != nil {
		return notification.Default(), err
	}

	return notification.New(notification.NewParams{
		ID:         uuid.New(),
		EvidenceID: cmd.EvidenceID,
		Kind:       cmd.Kind,
		Recipient:  recipient,
		CreatedAt:  time.Now().UTC(),
	})
}

func (uc *SendEvidenceNotification) deliver(ctx context.Context, cmd Command, stored *notification.Notification) error {
	rendered, err := uc.renderer.Render(stored.Kind(), ports.EmailContent{
		RecipientName: stored.Recipient().Name(),
		LotID:         cmd.LotID,
		LotCode:       cmd.LotCode,
		Stage:         cmd.Stage,
		FileName:      cmd.FileName,
		Reason:        cmd.Reason,
	})
	if err != nil {
		return exception.Wrap(err, "render email")
	}

	sendErr := uc.sender.Send(ctx, ports.Email{
		IdempotencyKey: stored.ID().String(),
		To:             stored.Recipient(),
		Subject:        rendered.Subject,
		Text:           rendered.Text,
		HTML:           rendered.HTML,
	})

	switch {
	case sendErr == nil:
		return uc.recordSent(ctx, cmd, stored)
	case exception.HasCode(sendErr, exception.CodeEmailRejected):
		return uc.recordRejected(ctx, cmd, stored, sendErr)
	default:
		uc.recordFailedAttemptBestEffort(ctx, stored, sendErr)
		return exception.Wrap(sendErr, "send email")
	}
}

func (uc *SendEvidenceNotification) recordSent(ctx context.Context, cmd Command, stored *notification.Notification) error {
	sent, err := stored.MarkSent(time.Now().UTC())
	if err != nil {
		return err
	}

	if err := uc.tx.WithinTx(ctx, func(ctx context.Context) error {
		return uc.updateAndRegister(ctx, cmd, sent)
	}); err != nil {
		return exception.Wrap(err, "record sent notification transaction")
	}
	return nil
}

func (uc *SendEvidenceNotification) recordRejected(ctx context.Context, cmd Command, stored *notification.Notification, sendErr error) error {
	failed, err := stored.MarkFailed(sendErr.Error())
	if err != nil {
		return err
	}

	if err := uc.tx.WithinTx(ctx, func(ctx context.Context) error {
		return uc.updateAndRegister(ctx, cmd, failed)
	}); err != nil {
		return exception.Wrap(err, "record rejected notification transaction")
	}
	return sendErr
}

func (uc *SendEvidenceNotification) updateAndRegister(ctx context.Context, cmd Command, updated *notification.Notification) error {
	if err := uc.log.Update(ctx, updated); err != nil {
		return exception.Wrap(err, "update notification")
	}
	return uc.registerProcessed(ctx, cmd)
}

func (uc *SendEvidenceNotification) markHandled(ctx context.Context, cmd Command) error {
	return uc.registerProcessed(ctx, cmd)
}

func (uc *SendEvidenceNotification) registerProcessed(ctx context.Context, cmd Command) error {
	if _, err := uc.processed.Register(ctx, cmd.MessageID, cmd.EventType); err != nil {
		return exception.Wrap(err, "register processed event")
	}
	return nil
}

func (uc *SendEvidenceNotification) recordFailedAttemptBestEffort(ctx context.Context, stored *notification.Notification, sendErr error) {
	attempted, err := stored.RecordFailedAttempt(sendErr.Error())
	if err != nil {
		return
	}
	_ = uc.log.Update(ctx, attempted)
}
