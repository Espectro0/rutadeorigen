package evidence_test

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"rutadeorigen/traceability/internal/domain/evidence"
	"rutadeorigen/traceability/internal/domain/lot"
	"rutadeorigen/traceability/internal/platform/exception"
)

const maxBytes = 5 << 20

func validParams() evidence.NewParams {
	return evidence.NewParams{
		LotID:       uuid.New(),
		Stage:       lot.StageFarm,
		UploadedBy:  uuid.New(),
		FileName:    "secado.jpg",
		ContentType: "image/jpeg",
		SizeBytes:   1024,
		MaxBytes:    maxBytes,
	}
}

func mustNew(t *testing.T) *evidence.Evidence {
	t.Helper()

	e, err := evidence.New(validParams())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return e
}

func uploaded(t *testing.T) *evidence.Evidence {
	t.Helper()

	e := mustNew(t)
	if err := e.MarkUploaded(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return e
}

func TestNewStartsPendingUpload(t *testing.T) {
	p := validParams()
	e, err := evidence.New(p)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if e.IsDefault() || e.Status() != evidence.StatusPendingUpload {
		t.Errorf("got default=%v status=%v, want a pending upload evidence", e.IsDefault(), e.Status())
	}
	if e.QuarantineKey() != p.LotID.String()+"/"+e.ID().String() {
		t.Errorf("the quarantine key must be built from identifiers only, got %q", e.QuarantineKey())
	}
	if !e.IsUploadedBy(p.UploadedBy) || e.IsUploadedBy(uuid.New()) || e.IsUploadedBy(uuid.Nil) {
		t.Error("only the uploader owns the evidence")
	}
}

func TestNewKeepsOnlyTheBaseNameOfTheFile(t *testing.T) {
	tests := map[string]string{
		"../../etc/passwd":     "passwd",
		`C:\fotos\secado.png`:  "secado.png",
		"  foto\x00 final.jpg": "foto final.jpg",
	}

	for in, want := range tests {
		p := validParams()
		p.FileName = in

		e, err := evidence.New(p)

		if err != nil {
			t.Fatalf("%q: unexpected error: %v", in, err)
		}
		if e.FileName() != want {
			t.Errorf("%q: got %q, want %q", in, e.FileName(), want)
		}
	}
}

func TestNewErrors(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*evidence.NewParams)
		want   exception.Code
	}{
		{"lot without stage", func(p *evidence.NewParams) { p.Stage = lot.StageNone }, exception.CodeEvidenceLotWithoutStage},
		{"unknown stage", func(p *evidence.NewParams) { p.Stage = lot.StageUnknown }, exception.CodeLotInvalidStage},
		{"empty name", func(p *evidence.NewParams) { p.FileName = "  " }, exception.CodeEvidenceInvalidFileName},
		{"name is only a dot", func(p *evidence.NewParams) { p.FileName = "." }, exception.CodeEvidenceInvalidFileName},
		{"pdf", func(p *evidence.NewParams) { p.ContentType = "application/pdf" }, exception.CodeEvidenceInvalidContentType},
		{"svg", func(p *evidence.NewParams) { p.ContentType = "image/svg+xml" }, exception.CodeEvidenceInvalidContentType},
		{"empty file", func(p *evidence.NewParams) { p.SizeBytes = 0 }, exception.CodeEvidenceEmptyFile},
		{"too large", func(p *evidence.NewParams) { p.SizeBytes = maxBytes + 1 }, exception.CodeEvidenceTooLarge},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := validParams()
			tt.mutate(&p)

			e, err := evidence.New(p)

			if !exception.HasCode(err, tt.want) {
				t.Fatalf("got %v, want code %q", err, tt.want)
			}
			if !e.IsDefault() {
				t.Error("a failed creation must return the default evidence")
			}
		})
	}
}

func TestNewAcceptsTheAllowedImageTypes(t *testing.T) {
	for _, ct := range []string{"image/jpeg", "IMAGE/PNG", " image/webp "} {
		p := validParams()
		p.ContentType = ct

		if _, err := evidence.New(p); err != nil {
			t.Errorf("%q: unexpected error: %v", ct, err)
		}
	}
}

func TestMarkUploadedIsIdempotentUntilTheResult(t *testing.T) {
	e := uploaded(t)

	if err := e.MarkUploaded(); err != nil || e.Status() != evidence.StatusUploaded {
		t.Errorf("got %v and status %v, want no error and uploaded", err, e.Status())
	}
}

func TestMarkProcessed(t *testing.T) {
	at := time.Date(2026, time.November, 2, 15, 30, 0, 0, time.UTC)
	e := uploaded(t)

	if err := e.MarkProcessed("abc123", "final/key", "thumb/key", at); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if e.Status() != evidence.StatusProcessed || e.SHA256() != "abc123" || e.FinalKey() != "final/key" ||
		e.ThumbnailKey() != "thumb/key" || !e.ProcessedAt().Equal(at) {
		t.Errorf("unexpected processed evidence: %+v", e)
	}
	if e.ProcessedAt().Hour() != 15 {
		t.Error("the time of the result must not be truncated to the day")
	}
}

func TestMarkProcessedErrors(t *testing.T) {
	at := time.Now().UTC()

	tests := []struct {
		name   string
		prepare func(*testing.T) *evidence.Evidence
		sha    string
		want   exception.Code
	}{
		{"still pending the upload", mustNew, "abc", exception.CodeEvidenceInvalidResult},
		{"without hash", uploaded, " ", exception.CodeEvidenceInvalidResult},
		{"already processed", func(t *testing.T) *evidence.Evidence {
			e := uploaded(t)
			if err := e.MarkProcessed("abc", "f", "t", at); err != nil {
				t.Fatal(err)
			}
			return e
		}, "abc", exception.CodeEvidenceAlreadyFinal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := tt.prepare(t)

			err := e.MarkProcessed(tt.sha, "f", "t", at)

			if !exception.HasCode(err, tt.want) {
				t.Errorf("got %v, want code %q", err, tt.want)
			}
		})
	}
}

func TestMarkRejected(t *testing.T) {
	at := time.Now().UTC()
	e := uploaded(t)

	if err := e.MarkRejected("not an image", at); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if e.Status() != evidence.StatusRejected || e.RejectionReason() != "not an image" {
		t.Errorf("unexpected rejected evidence: %+v", e)
	}

	if err := e.MarkRejected("again", at); !exception.HasCode(err, exception.CodeEvidenceAlreadyFinal) {
		t.Errorf("got %v, want already final", err)
	}
	if err := e.MarkUploaded(); !exception.HasCode(err, exception.CodeEvidenceAlreadyFinal) {
		t.Errorf("a rejected evidence cannot go back to uploaded, got %v", err)
	}
}

func TestMarkRejectedRequiresAReason(t *testing.T) {
	e := uploaded(t)

	if err := e.MarkRejected(strings.Repeat(" ", 3), time.Now()); !exception.HasCode(err, exception.CodeEvidenceInvalidResult) {
		t.Errorf("got %v, want invalid result", err)
	}
}

func TestDefaultEvidence(t *testing.T) {
	var none *evidence.Evidence

	if !evidence.Default().IsDefault() || !none.IsDefault() {
		t.Error("the default evidence and nil must be reported as default")
	}
	if evidence.ParseStatus("whatever") != evidence.StatusUnknown {
		t.Error("an unknown status must parse to unknown")
	}
}
