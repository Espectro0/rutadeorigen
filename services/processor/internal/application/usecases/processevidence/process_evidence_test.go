package processevidence

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"rutadeorigen/processor/internal/application/output/ports"
	"rutadeorigen/processor/internal/domain/picture"
	"rutadeorigen/processor/internal/domain/rejection"
	"rutadeorigen/processor/internal/platform/exception"
)

type fakeProcessedEvents struct {
	registered map[uuid.UUID]bool
	existsErr  error
}

func (f *fakeProcessedEvents) Exists(_ context.Context, id uuid.UUID) (bool, error) {
	return f.registered[id], f.existsErr
}

func (f *fakeProcessedEvents) Register(_ context.Context, id uuid.UUID, _ string) (bool, error) {
	if f.registered[id] {
		return false, nil
	}
	f.registered[id] = true
	return true, nil
}

type fakeQuarantine struct {
	object  ports.QuarantineObject
	getErr  error
	removed []string
}

func (f *fakeQuarantine) Get(_ context.Context, _ string, _ int64) (ports.QuarantineObject, error) {
	return f.object, f.getErr
}

func (f *fakeQuarantine) Remove(_ context.Context, key string) error {
	f.removed = append(f.removed, key)
	return nil
}

type fakeEvidenceStorage struct {
	stored map[string][]byte
	putErr error
}

func (f *fakeEvidenceStorage) Put(_ context.Context, key, _ string, data []byte) error {
	if f.putErr != nil {
		return f.putErr
	}
	f.stored[key] = data
	return nil
}

type fakeImages struct {
	info       picture.Info
	inspectErr error
	processErr error
}

func (f *fakeImages) Inspect(_ context.Context, _ []byte) (picture.Info, error) {
	return f.info, f.inspectErr
}

func (f *fakeImages) Process(_ context.Context, _ []byte, _ picture.Info, _ int) (ports.ProcessedImage, error) {
	return ports.ProcessedImage{Final: []byte("final"), Thumbnail: []byte("thumb")}, f.processErr
}

type fakeMessages struct{}

func (fakeMessages) Message(reason rejection.Reason) string { return "rechazada: " + reason.Code() }

type fakePublisher struct {
	events []ports.Event
}

func (f *fakePublisher) Publish(_ context.Context, event ports.Event) error {
	f.events = append(f.events, event)
	return nil
}

type fakeTx struct{}

func (fakeTx) WithinTx(ctx context.Context, fn func(ctx context.Context) error) error { return fn(ctx) }

type fixture struct {
	uc         *ProcessEvidence
	processed  *fakeProcessedEvents
	quarantine *fakeQuarantine
	evidences  *fakeEvidenceStorage
	images     *fakeImages
	publisher  *fakePublisher
}

var originalBytes = []byte("original-image-bytes")

func newFixture(t *testing.T) *fixture {
	t.Helper()
	info, err := picture.NewInfo("jpeg", 4000, 3000)
	if err != nil {
		t.Fatal(err)
	}
	limits, err := picture.NewLimits(5<<20, 40, 400)
	if err != nil {
		t.Fatal(err)
	}

	f := &fixture{
		processed:  &fakeProcessedEvents{registered: map[uuid.UUID]bool{}},
		quarantine: &fakeQuarantine{object: ports.QuarantineObject{Exists: true, Data: originalBytes}},
		evidences:  &fakeEvidenceStorage{stored: map[string][]byte{}},
		images:     &fakeImages{info: info},
		publisher:  &fakePublisher{},
	}
	f.uc = New(f.processed, f.quarantine, f.evidences, f.images, fakeMessages{}, f.publisher, fakeTx{}, limits)
	return f
}

func newCommand() Command {
	lotID, evidenceID := uuid.New(), uuid.New()
	return Command{
		MessageID:     uuid.New(),
		EventType:     "evidence.uploaded",
		EvidenceID:    evidenceID,
		LotID:         lotID,
		QuarantineKey: lotID.String() + "/" + evidenceID.String(),
		SizeBytes:     int64(len(originalBytes)),
		Uploader:      Uploader{ActorID: uuid.New(), Role: "producer"},
	}
}

func TestExecuteProcessesValidImage(t *testing.T) {
	f := newFixture(t)
	cmd := newCommand()

	if err := f.uc.Execute(context.Background(), cmd); err != nil {
		t.Fatal(err)
	}

	if len(f.evidences.stored) != 2 {
		t.Errorf("want final and thumbnail stored, got %d", len(f.evidences.stored))
	}
	if len(f.publisher.events) != 1 || f.publisher.events[0].Type != ports.EventEvidenceProcessed {
		t.Fatalf("want one evidence.processed event, got %+v", f.publisher.events)
	}
	payload, ok := f.publisher.events[0].Payload.(EvidenceProcessedPayload)
	if !ok || payload.SHA256 == "" || payload.EvidenceID != cmd.EvidenceID {
		t.Errorf("unexpected payload %+v", f.publisher.events[0].Payload)
	}
	if len(f.quarantine.removed) != 1 {
		t.Error("original must be removed from quarantine")
	}
	if !f.processed.registered[cmd.MessageID] {
		t.Error("message must be registered")
	}
}

func TestExecuteSkipsMessageAlreadyHandled(t *testing.T) {
	f := newFixture(t)
	cmd := newCommand()
	f.processed.registered[cmd.MessageID] = true
	f.quarantine.getErr = errors.New("must not be called")

	if err := f.uc.Execute(context.Background(), cmd); err != nil {
		t.Fatal(err)
	}
	if len(f.publisher.events) != 0 {
		t.Error("nothing must be published")
	}
}

func TestExecuteRejectsCorruptImage(t *testing.T) {
	f := newFixture(t)
	f.images.inspectErr = exception.New(exception.CodeImageCorrupt)

	if err := f.uc.Execute(context.Background(), newCommand()); err != nil {
		t.Fatal(err)
	}

	if len(f.publisher.events) != 1 || f.publisher.events[0].Type != ports.EventEvidenceRejected {
		t.Fatalf("want one evidence.rejected event, got %+v", f.publisher.events)
	}
	payload := f.publisher.events[0].Payload.(EvidenceRejectedPayload)
	if payload.Reason != "rechazada: "+string(exception.CodeImageCorrupt) {
		t.Errorf("reason = %q", payload.Reason)
	}
	if len(f.evidences.stored) != 0 {
		t.Error("nothing must be stored for a rejected file")
	}
}

func TestExecuteRejectsMissingFile(t *testing.T) {
	f := newFixture(t)
	f.quarantine.object = ports.QuarantineObject{}

	if err := f.uc.Execute(context.Background(), newCommand()); err != nil {
		t.Fatal(err)
	}
	if f.publisher.events[0].Type != ports.EventEvidenceRejected {
		t.Error("missing file must be rejected")
	}
}

func TestExecuteRejectsSizeMismatch(t *testing.T) {
	f := newFixture(t)
	f.quarantine.object = ports.QuarantineObject{Exists: true, Data: append(originalBytes, 'x')}

	if err := f.uc.Execute(context.Background(), newCommand()); err != nil {
		t.Fatal(err)
	}
	if f.publisher.events[0].Type != ports.EventEvidenceRejected {
		t.Error("size mismatch must be rejected")
	}
}

func TestExecuteRejectsTooManyPixels(t *testing.T) {
	f := newFixture(t)
	f.images.info, _ = picture.NewInfo("png", 7000, 6000)

	if err := f.uc.Execute(context.Background(), newCommand()); err != nil {
		t.Fatal(err)
	}
	if f.publisher.events[0].Type != ports.EventEvidenceRejected {
		t.Error("too many pixels must be rejected")
	}
}

func TestExecuteRejectsAnnouncedSizeAboveLimit(t *testing.T) {
	f := newFixture(t)
	cmd := newCommand()
	cmd.SizeBytes = 6 << 20

	if err := f.uc.Execute(context.Background(), cmd); err != nil {
		t.Fatal(err)
	}
	if f.publisher.events[0].Type != ports.EventEvidenceRejected {
		t.Error("oversized file must be rejected")
	}
}

func TestExecuteReturnsInfrastructureErrorsForRetry(t *testing.T) {
	cases := map[string]func(f *fixture){
		"quarantine read": func(f *fixture) { f.quarantine.getErr = exception.New(exception.CodeStorageUnavailable) },
		"evidence put":    func(f *fixture) { f.evidences.putErr = exception.New(exception.CodeStorageUnavailable) },
		"image process":   func(f *fixture) { f.images.processErr = exception.New(exception.CodeImageProcessingFailed) },
		"exists check":    func(f *fixture) { f.processed.existsErr = errors.New("db down") },
	}

	for name, breakIt := range cases {
		t.Run(name, func(t *testing.T) {
			f := newFixture(t)
			breakIt(f)

			if err := f.uc.Execute(context.Background(), newCommand()); err == nil {
				t.Fatal("want error so the message is retried")
			}
			if len(f.publisher.events) != 0 {
				t.Error("nothing must be published on failure")
			}
		})
	}
}

func TestExecuteReturnsErrorForKeyMismatch(t *testing.T) {
	f := newFixture(t)
	cmd := newCommand()
	cmd.QuarantineKey = uuid.NewString() + "/" + cmd.EvidenceID.String()

	err := f.uc.Execute(context.Background(), cmd)
	if !exception.HasCode(err, exception.CodeEvidenceKeyMismatch) {
		t.Errorf("want key mismatch, got %v", err)
	}
	if len(f.publisher.events) != 0 {
		t.Error("nothing must be published for a broken event")
	}
}

func TestExecuteDoesNotPublishTwiceWhenRegisteredConcurrently(t *testing.T) {
	f := newFixture(t)
	cmd := newCommand()
	f.processed.existsErr = nil
	registeredByAnotherWorker := &fakeProcessedEvents{registered: map[uuid.UUID]bool{}}
	f.uc.processed = &lateRegistration{inner: registeredByAnotherWorker, messageID: cmd.MessageID}

	if err := f.uc.Execute(context.Background(), cmd); err != nil {
		t.Fatal(err)
	}
	if len(f.publisher.events) != 0 {
		t.Error("an already registered message must not publish again")
	}
}

type lateRegistration struct {
	inner     *fakeProcessedEvents
	messageID uuid.UUID
}

func (l *lateRegistration) Exists(_ context.Context, _ uuid.UUID) (bool, error) { return false, nil }

func (l *lateRegistration) Register(ctx context.Context, id uuid.UUID, eventType string) (bool, error) {
	l.inner.registered[l.messageID] = true
	return l.inner.Register(ctx, id, eventType)
}
