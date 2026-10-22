package scan_test

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"rutadeorigen/qr/internal/domain/scan"
	"rutadeorigen/qr/internal/platform/exception"
)

func TestNewValid(t *testing.T) {
	lotID := uuid.New()

	s, err := scan.New(lotID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if s.LotID() != lotID {
		t.Error("unexpected lot ID")
	}
	if s.ScannedAt().IsZero() || s.IsDefault() {
		t.Error("expected a scan date and a non default scan")
	}
}

func TestNewInvalid(t *testing.T) {
	s, err := scan.New(uuid.Nil)

	if !exception.HasCode(err, exception.CodeLotViewInvalidID) {
		t.Errorf("got %v, want invalid lot ID", err)
	}
	if !s.IsDefault() {
		t.Error("expected default scan on error")
	}
}

func TestDefault(t *testing.T) {
	s := scan.Default()

	if !s.IsDefault() || !s.ScannedAt().IsZero() {
		t.Error("Default must be default with no date")
	}
	if scan.Default() == s {
		t.Error("Default must return a new instance each time")
	}
}

func TestIsDefaultOnNil(t *testing.T) {
	var s *scan.Scan

	if !s.IsDefault() {
		t.Error("a nil scan must be default")
	}
}

func TestRestore(t *testing.T) {
	lotID := uuid.New()
	at := time.Now().UTC().Add(-time.Hour)

	s := scan.Restore(lotID, at)

	if s.LotID() != lotID || !s.ScannedAt().Equal(at) {
		t.Error("fields were not restored")
	}
}
