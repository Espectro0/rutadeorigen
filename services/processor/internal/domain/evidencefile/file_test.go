package evidencefile

import (
	"testing"

	"github.com/google/uuid"

	"rutadeorigen/processor/internal/domain/picture"
	"rutadeorigen/processor/internal/platform/exception"
)

func params() NewParams {
	lot, ev := uuid.New(), uuid.New()
	return NewParams{EvidenceID: ev, LotID: lot, QuarantineKey: lot.String() + "/" + ev.String(), SizeBytes: 1000}
}

func TestNewValid(t *testing.T) {
	p := params()
	f, err := New(p)
	if err != nil || f.IsDefault() {
		t.Fatalf("got %v %v", f, err)
	}
	if f.MaxDownloadBytes() != 1001 {
		t.Error("cap")
	}
	if f.FinalKey(picture.FormatJPEG) != p.LotID.String()+"/"+p.EvidenceID.String()+"/imagen.jpg" {
		t.Error("final key")
	}
	if f.ThumbnailKey() != p.LotID.String()+"/"+p.EvidenceID.String()+"/miniatura.webp" {
		t.Error("thumbnail key")
	}
}

func TestNewInvalid(t *testing.T) {
	p := params()
	p.SizeBytes = 0
	if f, err := New(p); !exception.HasCode(err, exception.CodeEventInvalidPayload) || !f.IsDefault() {
		t.Errorf("size: %v", err)
	}

	p = params()
	p.LotID = uuid.Nil
	if _, err := New(p); !exception.HasCode(err, exception.CodeEventInvalidPayload) {
		t.Errorf("lot: %v", err)
	}

	p = params()
	p.QuarantineKey = uuid.NewString() + "/" + p.EvidenceID.String()
	if _, err := New(p); !exception.HasCode(err, exception.CodeEvidenceKeyMismatch) {
		t.Errorf("key: %v", err)
	}
}

func TestCheckUploadedSize(t *testing.T) {
	f, _ := New(params())
	if err := f.CheckUploadedSize(1000); err != nil {
		t.Error(err)
	}
	if err := f.CheckUploadedSize(999); !exception.HasCode(err, exception.CodeFileSizeMismatch) {
		t.Errorf("got %v", err)
	}
}
