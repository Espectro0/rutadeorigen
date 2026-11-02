package rejectionmessages

import (
	"strings"
	"testing"

	"rutadeorigen/processor/internal/domain/rejection"
	"rutadeorigen/processor/internal/platform/catalog"
	"rutadeorigen/processor/internal/platform/exception"
)

func newMessages(t *testing.T) *Catalog {
	t.Helper()
	processorCatalog, err := catalog.New()
	if err != nil {
		t.Fatal(err)
	}
	return NewCatalog(processorCatalog)
}

func TestMessageResolvesSpanishText(t *testing.T) {
	reason := rejection.FromError(exception.New(exception.CodeImageCorrupt))

	if got := newMessages(t).Message(reason); !strings.Contains(got, "dañada") {
		t.Errorf("unexpected message %q", got)
	}
}

func TestMessageFillsParameters(t *testing.T) {
	reason := rejection.FromError(exception.New(exception.CodeFileTooLarge).WithParams(map[string]any{"max": 5}))

	if got := newMessages(t).Message(reason); !strings.Contains(got, "5 MB") {
		t.Errorf("unexpected message %q", got)
	}
}

func TestMessageIsEmptyForDefaultReason(t *testing.T) {
	if got := newMessages(t).Message(rejection.Default()); got != "" {
		t.Errorf("default reason must give an empty message, got %q", got)
	}
}
