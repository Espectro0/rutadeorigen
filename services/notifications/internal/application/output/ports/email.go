package ports

import (
	"context"

	"github.com/google/uuid"

	"rutadeorigen/notifications/internal/domain/notification"
)

type EmailContent struct {
	RecipientName string
	LotID         uuid.UUID
	LotCode       string
	Stage         string
	FileName      string
	Reason        string
}

type RenderedEmail struct {
	Subject string
	Text    string
	HTML    string
}

type Email struct {
	IdempotencyKey string
	To             notification.Recipient
	Subject        string
	Text           string
	HTML           string
}

type EmailRenderer interface {
	Render(kind notification.Kind, content EmailContent) (RenderedEmail, error)
}

type EmailSender interface {
	Send(ctx context.Context, email Email) error
}
