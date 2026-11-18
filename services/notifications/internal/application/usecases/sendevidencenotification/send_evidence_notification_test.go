package sendevidencenotification

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"rutadeorigen/notifications/internal/application/output/ports"
	"rutadeorigen/notifications/internal/domain/notification"
	"rutadeorigen/notifications/internal/platform/exception"
)

type fakeProcessedEvents struct {
	registered  map[uuid.UUID]bool
	existsErr   error
	registerErr error
}

func (f *fakeProcessedEvents) Exists(_ context.Context, id uuid.UUID) (bool, error) {
	return f.registered[id], f.existsErr
}

func (f *fakeProcessedEvents) Register(_ context.Context, id uuid.UUID, _ string) (bool, error) {
	if f.registerErr != nil {
		return false, f.registerErr
	}
	if f.registered[id] {
		return false, nil
	}
	f.registered[id] = true
	return true, nil
}

type logKey struct {
	evidenceID uuid.UUID
	kind       notification.Kind
}

type fakeLog struct {
	stored     map[logKey]*notification.Notification
	reserveErr error
	updateErr  error
	updates    []*notification.Notification
}

func (f *fakeLog) Reserve(_ context.Context, candidate *notification.Notification) (*notification.Notification, error) {
	if f.reserveErr != nil {
		return notification.Default(), f.reserveErr
	}
	key := logKey{candidate.EvidenceID(), candidate.Kind()}
	if existing, ok := f.stored[key]; ok {
		return existing, nil
	}
	f.stored[key] = candidate
	return candidate, nil
}

func (f *fakeLog) Update(_ context.Context, updated *notification.Notification) error {
	if f.updateErr != nil {
		return f.updateErr
	}
	f.updates = append(f.updates, updated)
	f.stored[logKey{updated.EvidenceID(), updated.Kind()}] = updated
	return nil
}

type fakeRenderer struct {
	err     error
	kind    notification.Kind
	content ports.EmailContent
}

func (f *fakeRenderer) Render(kind notification.Kind, content ports.EmailContent) (ports.RenderedEmail, error) {
	f.kind, f.content = kind, content
	if f.err != nil {
		return ports.RenderedEmail{}, f.err
	}
	return ports.RenderedEmail{Subject: "subject", Text: "text", HTML: "<p>html</p>"}, nil
}

type fakeSender struct {
	err  error
	sent []ports.Email
}

func (f *fakeSender) Send(_ context.Context, email ports.Email) error {
	if f.err != nil {
		return f.err
	}
	f.sent = append(f.sent, email)
	return nil
}

type fakeTx struct{}

func (fakeTx) WithinTx(ctx context.Context, fn func(ctx context.Context) error) error { return fn(ctx) }

type fixture struct {
	uc        *SendEvidenceNotification
	processed *fakeProcessedEvents
	log       *fakeLog
	renderer  *fakeRenderer
	sender    *fakeSender
}

func newFixture() *fixture {
	f := &fixture{
		processed: &fakeProcessedEvents{registered: map[uuid.UUID]bool{}},
		log:       &fakeLog{stored: map[logKey]*notification.Notification{}},
		renderer:  &fakeRenderer{},
		sender:    &fakeSender{},
	}
	f.uc = New(f.processed, f.log, f.renderer, f.sender, fakeTx{})
	return f
}

func newCommand() Command {
	return Command{
		MessageID:      uuid.New(),
		EventType:      "evidence.processed",
		Kind:           notification.KindEvidenceProcessed,
		EvidenceID:     uuid.New(),
		LotID:          uuid.New(),
		LotCode:        "LT-001",
		Stage:          "farm",
		FileName:       "foto.jpg",
		RecipientName:  "Maria Lopez",
		RecipientEmail: "Maria@Example.com",
	}
}

func (f *fixture) storedFor(cmd Command) *notification.Notification {
	return f.log.stored[logKey{cmd.EvidenceID, cmd.Kind}]
}

func TestExecuteSendsAndRecordsTheNotification(t *testing.T) {
	f := newFixture()
	cmd := newCommand()

	if err := f.uc.Execute(context.Background(), cmd); err != nil {
		t.Fatal(err)
	}

	if len(f.sender.sent) != 1 {
		t.Fatalf("want one email, got %d", len(f.sender.sent))
	}
	email := f.sender.sent[0]
	stored := f.storedFor(cmd)
	if email.To.Email() != "maria@example.com" || email.Subject != "subject" || email.Text != "text" || email.HTML != "<p>html</p>" {
		t.Errorf("unexpected email %+v", email)
	}
	if email.IdempotencyKey != stored.ID().String() {
		t.Errorf("the idempotency key must be the notification id, got %q", email.IdempotencyKey)
	}
	if !stored.IsSent() || stored.Attempts() != 1 {
		t.Errorf("the notification must be sent after one attempt: %+v", stored)
	}
	if !f.processed.registered[cmd.MessageID] {
		t.Error("the message must be registered")
	}
}

func TestExecuteRendersWithTheEventData(t *testing.T) {
	f := newFixture()
	cmd := newCommand()
	cmd.Kind = notification.KindEvidenceRejected
	cmd.Reason = "La imagen está dañada"

	if err := f.uc.Execute(context.Background(), cmd); err != nil {
		t.Fatal(err)
	}

	want := ports.EmailContent{
		RecipientName: "Maria Lopez", LotID: cmd.LotID, LotCode: "LT-001",
		Stage: "farm", FileName: "foto.jpg", Reason: "La imagen está dañada",
	}
	if f.renderer.kind != notification.KindEvidenceRejected || f.renderer.content != want {
		t.Errorf("got kind %q and content %+v", f.renderer.kind, f.renderer.content)
	}
}

func TestExecuteSkipsMessageAlreadyHandled(t *testing.T) {
	f := newFixture()
	cmd := newCommand()
	f.processed.registered[cmd.MessageID] = true

	if err := f.uc.Execute(context.Background(), cmd); err != nil {
		t.Fatal(err)
	}

	if len(f.sender.sent) != 0 || len(f.log.stored) != 0 {
		t.Error("nothing must be reserved or sent")
	}
}

func TestExecuteDoesNotSendTwiceForTheSameEvidenceAndKind(t *testing.T) {
	f := newFixture()
	first := newCommand()
	if err := f.uc.Execute(context.Background(), first); err != nil {
		t.Fatal(err)
	}
	second := first
	second.MessageID = uuid.New()

	if err := f.uc.Execute(context.Background(), second); err != nil {
		t.Fatal(err)
	}

	if len(f.sender.sent) != 1 {
		t.Errorf("want a single email, got %d", len(f.sender.sent))
	}
	if !f.processed.registered[second.MessageID] {
		t.Error("the repeated message must be registered so it is not evaluated again")
	}
}

func TestExecuteDoesNotResendAFailedNotification(t *testing.T) {
	f := newFixture()
	cmd := newCommand()
	f.sender.err = exception.New(exception.CodeEmailRejected)
	_ = f.uc.Execute(context.Background(), cmd)
	f.sender.err = nil
	replay := cmd
	replay.MessageID = uuid.New()

	if err := f.uc.Execute(context.Background(), replay); err != nil {
		t.Fatal(err)
	}

	if len(f.sender.sent) != 0 {
		t.Error("a failed notification must not be sent again")
	}
}

func TestExecuteRetriesWithTheSameIdempotencyKey(t *testing.T) {
	f := newFixture()
	cmd := newCommand()
	f.sender.err = exception.New(exception.CodeEmailUnavailable)

	err := f.uc.Execute(context.Background(), cmd)

	if !exception.HasCode(err, exception.CodeEmailUnavailable) {
		t.Fatalf("want an unavailable error to retry, got %v", err)
	}
	stored := f.storedFor(cmd)
	if !stored.IsPending() || stored.Attempts() != 1 || stored.LastError() == "" {
		t.Errorf("the failed attempt must be recorded and stay pending: %+v", stored)
	}
	if f.processed.registered[cmd.MessageID] {
		t.Error("the message must not be registered while it can still be retried")
	}

	f.sender.err = nil
	if err := f.uc.Execute(context.Background(), cmd); err != nil {
		t.Fatal(err)
	}
	if len(f.sender.sent) != 1 || f.sender.sent[0].IdempotencyKey != stored.ID().String() {
		t.Errorf("the retry must reuse the notification id: %+v", f.sender.sent)
	}
	if !f.storedFor(cmd).IsSent() || !f.processed.registered[cmd.MessageID] {
		t.Error("the retry must complete the notification")
	}
}

func TestExecuteMarksRejectedEmailAsFailedAndReturnsThePermanentError(t *testing.T) {
	f := newFixture()
	cmd := newCommand()
	f.sender.err = exception.New(exception.CodeEmailRejected).WithTechnical("resend 422")

	err := f.uc.Execute(context.Background(), cmd)

	if !exception.HasCode(err, exception.CodeEmailRejected) {
		t.Fatalf("want the rejection error, got %v", err)
	}
	stored := f.storedFor(cmd)
	if !stored.IsFailed() || stored.LastError() == "" {
		t.Errorf("the notification must be failed with its reason: %+v", stored)
	}
	if !f.processed.registered[cmd.MessageID] {
		t.Error("the message must be registered so a replay does nothing")
	}
}

func TestExecuteRejectsAnInvalidRecipientWithoutSending(t *testing.T) {
	f := newFixture()
	cmd := newCommand()
	cmd.RecipientEmail = "not-an-email"

	err := f.uc.Execute(context.Background(), cmd)

	if !exception.HasCode(err, exception.CodeInvalidRecipient) {
		t.Fatalf("want an invalid recipient error, got %v", err)
	}
	if len(f.log.stored) != 0 || len(f.sender.sent) != 0 {
		t.Error("nothing must be reserved or sent")
	}
}

func TestExecuteRejectsAnInvalidKind(t *testing.T) {
	f := newFixture()
	cmd := newCommand()
	cmd.Kind = notification.KindUnknown

	err := f.uc.Execute(context.Background(), cmd)

	if !exception.HasCode(err, exception.CodeEventInvalidPayload) {
		t.Fatalf("want an invalid payload error, got %v", err)
	}
	if len(f.sender.sent) != 0 {
		t.Error("nothing must be sent")
	}
}

func TestExecuteReturnsInfrastructureErrors(t *testing.T) {
	boom := errors.New("db down")
	tests := map[string]func(*fixture){
		"exists":   func(f *fixture) { f.processed.existsErr = boom },
		"reserve":  func(f *fixture) { f.log.reserveErr = boom },
		"render":   func(f *fixture) { f.renderer.err = boom },
		"update":   func(f *fixture) { f.log.updateErr = boom },
		"register": func(f *fixture) { f.processed.registerErr = boom },
	}

	for name, setup := range tests {
		t.Run(name, func(t *testing.T) {
			f := newFixture()
			setup(f)

			err := f.uc.Execute(context.Background(), newCommand())

			if !errors.Is(err, boom) {
				t.Errorf("want the infrastructure error, got %v", err)
			}
			if exception.HasCode(err, exception.CodeEmailRejected) {
				t.Error("an infrastructure error must not look like a permanent rejection")
			}
		})
	}
}

func TestExecuteDoesNotMarkHandledWhenRecordingTheSendFails(t *testing.T) {
	f := newFixture()
	cmd := newCommand()
	f.log.updateErr = errors.New("db down")

	err := f.uc.Execute(context.Background(), cmd)

	if err == nil {
		t.Fatal("want an error so the message is retried")
	}
	if f.processed.registered[cmd.MessageID] {
		t.Error("the message must stay unregistered so the retry can finish it")
	}
	if len(f.sender.sent) != 1 {
		t.Errorf("the email was already sent once, got %d", len(f.sender.sent))
	}
}
