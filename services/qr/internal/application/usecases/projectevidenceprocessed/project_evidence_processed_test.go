package projectevidenceprocessed_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"rutadeorigen/qr/internal/application/usecases/projectevidenceprocessed"
	"rutadeorigen/qr/internal/domain/evidenceview"
	"rutadeorigen/qr/internal/platform/exception"
)

var errBackend = errors.New("backend is down")

type fakeTx struct{ calls int }

func (t *fakeTx) WithinTx(ctx context.Context, fn func(ctx context.Context) error) error {
	t.calls++
	return fn(ctx)
}

type fakeProcessed struct {
	seen map[uuid.UUID]bool
	err  error
}

func (p *fakeProcessed) Exists(_ context.Context, id uuid.UUID) (bool, error) { return p.seen[id], nil }

func (p *fakeProcessed) Register(_ context.Context, id uuid.UUID, _ string) (bool, error) {
	if p.err != nil {
		return false, p.err
	}
	if p.seen[id] {
		return false, nil
	}
	p.seen[id] = true
	return true, nil
}

type fakeEvidences struct {
	saved []*evidenceview.EvidenceView
	err   error
}

func (e *fakeEvidences) Save(_ context.Context, view *evidenceview.EvidenceView) error {
	if e.err != nil {
		return e.err
	}
	e.saved = append(e.saved, view)
	return nil
}

func (e *fakeEvidences) ListByLotID(context.Context, uuid.UUID, int) ([]*evidenceview.EvidenceView, error) {
	return nil, nil
}

type fixture struct {
	uc        *projectevidenceprocessed.ProjectEvidenceProcessed
	tx        *fakeTx
	processed *fakeProcessed
	evidences *fakeEvidences
}

func newFixture() fixture {
	f := fixture{tx: &fakeTx{}, processed: &fakeProcessed{seen: map[uuid.UUID]bool{}}, evidences: &fakeEvidences{}}
	f.uc = projectevidenceprocessed.New(f.evidences, f.processed, f.tx)
	return f
}

func validCommand() projectevidenceprocessed.Command {
	return projectevidenceprocessed.Command{
		MessageID:    uuid.New(),
		EventType:    "evidence.processed",
		EvidenceID:   uuid.New(),
		LotID:        uuid.New(),
		Stage:        "farm",
		FinalKey:     "lot/ev/imagen.jpg",
		ThumbnailKey: "lot/ev/miniatura.webp",
		ContentType:  "image/jpeg",
	}
}

func TestExecuteSavesTheEvidenceOnce(t *testing.T) {
	f := newFixture()
	cmd := validCommand()

	for attempt := 0; attempt < 2; attempt++ {
		if err := f.uc.Execute(context.Background(), cmd); err != nil {
			t.Fatal(err)
		}
	}

	if len(f.evidences.saved) != 1 || f.evidences.saved[0].EvidenceID() != cmd.EvidenceID {
		t.Errorf("want exactly one saved view, got %+v", f.evidences.saved)
	}
}

func TestExecuteRejectsInvalidCommands(t *testing.T) {
	cases := map[string]func(c *projectevidenceprocessed.Command){
		"no message id": func(c *projectevidenceprocessed.Command) { c.MessageID = uuid.Nil },
		"no event type": func(c *projectevidenceprocessed.Command) { c.EventType = "" },
		"no lot id":     func(c *projectevidenceprocessed.Command) { c.LotID = uuid.Nil },
		"bad stage":     func(c *projectevidenceprocessed.Command) { c.Stage = "mystery" },
		"bad type":      func(c *projectevidenceprocessed.Command) { c.ContentType = "text/plain" },
	}

	for name, breakIt := range cases {
		t.Run(name, func(t *testing.T) {
			f := newFixture()
			cmd := validCommand()
			breakIt(&cmd)

			err := f.uc.Execute(context.Background(), cmd)

			if !exception.HasCode(err, exception.CodeEventInvalidPayload) {
				t.Errorf("want invalid payload, got %v", err)
			}
			if f.tx.calls != 0 || len(f.evidences.saved) != 0 {
				t.Error("nothing must run for an invalid command")
			}
		})
	}
}

func TestExecuteReturnsInfrastructureErrors(t *testing.T) {
	cases := map[string]func(f fixture){
		"register": func(f fixture) { f.processed.err = errBackend },
		"save":     func(f fixture) { f.evidences.err = errBackend },
	}

	for name, breakIt := range cases {
		t.Run(name, func(t *testing.T) {
			f := newFixture()
			breakIt(f)

			err := f.uc.Execute(context.Background(), validCommand())

			if !errors.Is(err, errBackend) {
				t.Errorf("want the backend error, got %v", err)
			}
		})
	}
}
