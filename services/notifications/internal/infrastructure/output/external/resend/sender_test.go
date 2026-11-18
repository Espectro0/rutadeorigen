package resend

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"rutadeorigen/notifications/internal/application/output/ports"
	"rutadeorigen/notifications/internal/domain/notification"
	"rutadeorigen/notifications/internal/platform/exception"
)

type captured struct {
	auth           string
	idempotencyKey string
	path           string
	body           sendRequest
}

func newServer(t *testing.T, status int, response string, got *captured) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got.auth = r.Header.Get("Authorization")
		got.idempotencyKey = r.Header.Get("Idempotency-Key")
		got.path = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&got.body)
		w.WriteHeader(status)
		_, _ = io.WriteString(w, response)
	}))
	t.Cleanup(server.Close)
	return server
}

func newEmail(t *testing.T) ports.Email {
	t.Helper()

	recipient, err := notification.NewRecipient("Maria Lopez", "maria@example.com")
	if err != nil {
		t.Fatal(err)
	}
	return ports.Email{
		IdempotencyKey: uuid.NewString(),
		To:             recipient,
		Subject:        "Asunto",
		Text:           "texto",
		HTML:           "<p>html</p>",
	}
}

func newSender(server *httptest.Server) *Sender {
	return NewWithClient(Config{APIKey: "re_test", From: "noreply@rutadeorigen.co", BaseURL: server.URL + "/"},
		server.Client(), slog.New(slog.NewJSONHandler(io.Discard, nil)))
}

func TestSendPostsTheEmailToResend(t *testing.T) {
	var got captured
	server := newServer(t, http.StatusOK, `{"id":"abc"}`, &got)
	email := newEmail(t)

	if err := newSender(server).Send(context.Background(), email); err != nil {
		t.Fatal(err)
	}

	if got.path != "/emails" || got.auth != "Bearer re_test" || got.idempotencyKey != email.IdempotencyKey {
		t.Errorf("unexpected request %+v", got)
	}
	if got.body.From != `"Ori - Ruta de Origen" <noreply@rutadeorigen.co>` {
		t.Errorf("from = %q", got.body.From)
	}
	if len(got.body.To) != 1 || got.body.To[0] != `"Maria Lopez" <maria@example.com>` {
		t.Errorf("to = %v", got.body.To)
	}
	if got.body.Subject != "Asunto" || got.body.HTML != "<p>html</p>" || got.body.Text != "texto" || got.body.ReplyTo != "noreply@rutadeorigen.co" {
		t.Errorf("unexpected body %+v", got.body)
	}
}

func TestSendClassifiesResponses(t *testing.T) {
	tests := map[string]struct {
		status int
		want   exception.Code
	}{
		"validation error":  {http.StatusUnprocessableEntity, exception.CodeEmailRejected},
		"bad request":       {http.StatusBadRequest, exception.CodeEmailRejected},
		"invalid api key":   {http.StatusUnauthorized, exception.CodeEmailUnavailable},
		"unverified domain": {http.StatusForbidden, exception.CodeEmailUnavailable},
		"idempotency race":  {http.StatusConflict, exception.CodeEmailUnavailable},
		"rate limited":      {http.StatusTooManyRequests, exception.CodeEmailUnavailable},
		"server error":      {http.StatusBadGateway, exception.CodeEmailUnavailable},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var got captured
			server := newServer(t, tc.status, `{"message":"nope"}`, &got)

			err := newSender(server).Send(context.Background(), newEmail(t))

			if !exception.HasCode(err, tc.want) {
				t.Errorf("want %s, got %v", tc.want, err)
			}
		})
	}
}

func TestSendReportsUnreachableProviderAsUnavailable(t *testing.T) {
	var got captured
	server := newServer(t, http.StatusOK, `{}`, &got)
	sender := newSender(server)
	server.Close()

	err := sender.Send(context.Background(), newEmail(t))

	if !exception.HasCode(err, exception.CodeEmailUnavailable) {
		t.Errorf("want unavailable, got %v", err)
	}
}

func TestSendReportsCancelledContextAsUnavailable(t *testing.T) {
	var got captured
	server := newServer(t, http.StatusOK, `{}`, &got)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := newSender(server).Send(ctx, newEmail(t))

	if !exception.HasCode(err, exception.CodeEmailUnavailable) {
		t.Errorf("want unavailable, got %v", err)
	}
}

func TestNewWithClientAppliesDefaults(t *testing.T) {
	sender := NewWithClient(Config{APIKey: "k", From: "a@b.co"}, http.DefaultClient, slog.New(slog.NewJSONHandler(io.Discard, nil)))

	if sender.cfg.BaseURL != DefaultBaseURL || sender.cfg.SenderName != DefaultSenderName {
		t.Errorf("unexpected defaults %+v", sender.cfg)
	}
}
