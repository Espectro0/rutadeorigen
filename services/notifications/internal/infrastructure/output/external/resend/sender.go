package resend

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"rutadeorigen/lib/resilience"
	"rutadeorigen/notifications/internal/application/output/ports"
	"rutadeorigen/notifications/internal/platform/exception"
)

var _ ports.EmailSender = (*Sender)(nil)

const (
	DefaultBaseURL    = "https://api.resend.com"
	DefaultSenderName = "Ori - Ruta de Origen"

	emailsPath       = "/emails"
	requestTimeout   = 10 * time.Second
	maxResponseBytes = 64 << 10
	breakerName      = "resend"
	idempotencyKey   = "Idempotency-Key"
)

var retryableClientStatuses = map[int]bool{
	http.StatusUnauthorized:    true,
	http.StatusForbidden:       true,
	http.StatusRequestTimeout:  true,
	http.StatusConflict:        true,
	http.StatusTooManyRequests: true,
}

type Config struct {
	APIKey     string
	From       string
	SenderName string
	BaseURL    string
}

type Sender struct {
	cfg    Config
	client *http.Client
	log    *slog.Logger
}

func New(cfg Config, log *slog.Logger) *Sender {
	client := &http.Client{
		Timeout:   requestTimeout,
		Transport: resilience.NewTransport(resilience.NewBreaker(breakerName, log), http.DefaultTransport),
	}
	return NewWithClient(cfg, client, log)
}

func NewWithClient(cfg Config, client *http.Client, log *slog.Logger) *Sender {
	if strings.TrimSpace(cfg.BaseURL) == "" {
		cfg.BaseURL = DefaultBaseURL
	}
	cfg.BaseURL = strings.TrimRight(cfg.BaseURL, "/")
	if strings.TrimSpace(cfg.SenderName) == "" {
		cfg.SenderName = DefaultSenderName
	}
	return &Sender{cfg: cfg, client: client, log: log}
}

type sendRequest struct {
	From    string            `json:"from"`
	To      []string          `json:"to"`
	Subject string            `json:"subject"`
	HTML    string            `json:"html"`
	Text    string            `json:"text"`
	ReplyTo string            `json:"reply_to,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
}

type sendResponse struct {
	ID      string `json:"id"`
	Message string `json:"message"`
}

func (s *Sender) Send(ctx context.Context, email ports.Email) error {
	body, err := json.Marshal(s.payload(email))
	if err != nil {
		return exception.New(exception.CodeInternal).WithTechnical("encode resend payload").WithCause(err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.cfg.BaseURL+emailsPath, bytes.NewReader(body))
	if err != nil {
		return exception.New(exception.CodeInternal).WithTechnical("build resend request").WithCause(err)
	}
	req.Header.Set("Authorization", "Bearer "+s.cfg.APIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(idempotencyKey, email.IdempotencyKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return exception.New(exception.CodeEmailUnavailable).WithTechnical("call resend").WithCause(err)
	}
	defer func() { _ = resp.Body.Close() }()

	raw, _ := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	var parsed sendResponse
	_ = json.Unmarshal(raw, &parsed)

	switch {
	case resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices:
		s.log.Info("email sent", "provider_id", parsed.ID, "idempotency_key", email.IdempotencyKey)
		return nil
	case resp.StatusCode >= http.StatusInternalServerError || retryableClientStatuses[resp.StatusCode]:
		return exception.New(exception.CodeEmailUnavailable).WithTechnical(describe(resp.StatusCode, parsed.Message))
	default:
		return exception.New(exception.CodeEmailRejected).WithTechnical(describe(resp.StatusCode, parsed.Message))
	}
}

func (s *Sender) payload(email ports.Email) sendRequest {
	from := mail.Address{Name: s.cfg.SenderName, Address: s.cfg.From}
	to := mail.Address{Name: email.To.Name(), Address: email.To.Email()}

	return sendRequest{
		From:    from.String(),
		To:      []string{to.String()},
		Subject: email.Subject,
		HTML:    email.HTML,
		Text:    email.Text,
		ReplyTo: s.cfg.From,
		Headers: map[string]string{
			"Importance":        "high",
			"X-Priority":        "1",
			"X-MSMail-Priority": "High",
		},
	}
}

func describe(status int, message string) string {
	return fmt.Sprintf("resend responded %d: %s", status, strings.TrimSpace(message))
}
