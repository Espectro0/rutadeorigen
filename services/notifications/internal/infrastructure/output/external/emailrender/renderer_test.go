package emailrender

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"rutadeorigen/notifications/internal/application/output/ports"
	"rutadeorigen/notifications/internal/domain/notification"
	"rutadeorigen/notifications/internal/platform/exception"
)

func content() ports.EmailContent {
	return ports.EmailContent{
		RecipientName: "Maria",
		LotID:         uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		LotCode:       "LT-001",
		Stage:         "roasting",
		FileName:      "foto.jpg",
		Reason:        "El archivo no es una imagen",
	}
}

func newRenderer(t *testing.T) *Renderer {
	t.Helper()

	renderer, err := New(" https://rutadeorigen.co/ ")
	if err != nil {
		t.Fatal(err)
	}
	return renderer
}

func TestRenderProcessedEmail(t *testing.T) {
	email, err := newRenderer(t).Render(notification.KindEvidenceProcessed, content())

	if err != nil {
		t.Fatal(err)
	}
	if email.Subject != "Tu foto del lote LT-001 ya está publicada" {
		t.Errorf("subject = %q", email.Subject)
	}
	for _, part := range []string{email.HTML, email.Text} {
		for _, want := range []string{"¡Hola, Maria!", "«foto.jpg»", "LT-001", "Tostión", "https://rutadeorigen.co/lotes/11111111-1111-1111-1111-111111111111"} {
			if !strings.Contains(part, want) {
				t.Errorf("missing %q in %q", want, part)
			}
		}
	}
}

func TestRenderRejectedEmailIncludesTheReason(t *testing.T) {
	email, err := newRenderer(t).Render(notification.KindEvidenceRejected, content())

	if err != nil {
		t.Fatal(err)
	}
	if email.Subject != "No pudimos procesar tu foto del lote LT-001" {
		t.Errorf("subject = %q", email.Subject)
	}
	if !strings.Contains(email.HTML, "El archivo no es una imagen") || !strings.Contains(email.Text, "El archivo no es una imagen") {
		t.Error("the reason must be included")
	}
}

func TestRenderOmitsMissingOptionalFields(t *testing.T) {
	c := content()
	c.RecipientName, c.FileName, c.Reason, c.Stage = "", "", "", "unknown-stage"

	email, err := newRenderer(t).Render(notification.KindEvidenceRejected, c)

	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(email.Text, "¡Hola!") || !strings.Contains(email.Text, "tu foto del lote") ||
		!strings.Contains(email.Text, "unknown-stage") || strings.Contains(email.Text, "Motivo") {
		t.Errorf("unexpected text %q", email.Text)
	}
}

func TestRenderEscapesHTML(t *testing.T) {
	c := content()
	c.RecipientName = "<script>alert(1)</script>"

	email, err := newRenderer(t).Render(notification.KindEvidenceProcessed, c)

	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(email.HTML, "<script>") {
		t.Error("html must be escaped")
	}
}

func TestRenderRejectsUnknownKind(t *testing.T) {
	_, err := newRenderer(t).Render(notification.KindUnknown, content())

	if !exception.HasCode(err, exception.CodeEmailRenderFailed) {
		t.Errorf("want render failed, got %v", err)
	}
}
