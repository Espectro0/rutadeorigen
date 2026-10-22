package qrcode_test

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"rutadeorigen/qr/internal/domain/qrcode"
	"rutadeorigen/qr/internal/platform/exception"
)

const baseURL = "https://rutadeorigen.co"

func TestNewValid(t *testing.T) {
	lotID := uuid.New()

	q, err := qrcode.New(lotID, baseURL, qrcode.ErrorCorrectionQuartile, "  v1.2.3  ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if q.LotID() != lotID {
		t.Error("unexpected lot ID")
	}
	if want := baseURL + "/l/" + lotID.String(); q.Content() != want {
		t.Errorf("Content: got %q, want %q", q.Content(), want)
	}
	if q.ErrorCorrection() != qrcode.ErrorCorrectionQuartile || q.LibraryVersion() != "v1.2.3" {
		t.Error("unexpected error correction or library version")
	}
	if q.CreatedAt().IsZero() || q.IsDefault() {
		t.Error("expected a creation date and a non default QR code")
	}
}

func TestNewNormalizesBaseURL(t *testing.T) {
	lotID := uuid.New()

	tests := []struct {
		name string
		base string
		want string
	}{
		{"trailing slash", "https://rutadeorigen.co/", "https://rutadeorigen.co/l/" + lotID.String()},
		{"surrounding spaces", "  https://rutadeorigen.co  ", "https://rutadeorigen.co/l/" + lotID.String()},
		{"with path", "https://rutadeorigen.co/api/", "https://rutadeorigen.co/api/l/" + lotID.String()},
		{"http with port", "http://localhost:8080", "http://localhost:8080/l/" + lotID.String()},
		{"query is dropped", "https://rutadeorigen.co?x=1", "https://rutadeorigen.co/l/" + lotID.String()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, err := qrcode.New(lotID, tt.base, qrcode.ErrorCorrectionQuartile, "v1")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if q.Content() != tt.want {
				t.Errorf("Content: got %q, want %q", q.Content(), tt.want)
			}
		})
	}
}

func TestNewInvalid(t *testing.T) {
	tests := []struct {
		name            string
		lotID           uuid.UUID
		baseURL         string
		errorCorrection qrcode.ErrorCorrection
		libraryVersion  string
		wantCode        exception.Code
	}{
		{"default lot ID", uuid.Nil, baseURL, qrcode.ErrorCorrectionQuartile, "v1", exception.CodeLotViewInvalidID},
		{"empty base URL", uuid.New(), "", qrcode.ErrorCorrectionQuartile, "v1", exception.CodeQRCodeInvalidContent},
		{"base URL without scheme", uuid.New(), "rutadeorigen.co", qrcode.ErrorCorrectionQuartile, "v1", exception.CodeQRCodeInvalidContent},
		{"base URL with other scheme", uuid.New(), "ftp://rutadeorigen.co", qrcode.ErrorCorrectionQuartile, "v1", exception.CodeQRCodeInvalidContent},
		{"base URL without host", uuid.New(), "https://", qrcode.ErrorCorrectionQuartile, "v1", exception.CodeQRCodeInvalidContent},
		{"malformed base URL", uuid.New(), "https://rutadeorigen.co/%zz", qrcode.ErrorCorrectionQuartile, "v1", exception.CodeQRCodeInvalidContent},
		{"empty library version", uuid.New(), baseURL, qrcode.ErrorCorrectionQuartile, "   ", exception.CodeQRCodeInvalidContent},
		{"unknown error correction", uuid.New(), baseURL, qrcode.ErrorCorrectionUnknown, "v1", exception.CodeQRCodeInvalidErrorCorrection},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, err := qrcode.New(tt.lotID, tt.baseURL, tt.errorCorrection, tt.libraryVersion)
			if !exception.HasCode(err, tt.wantCode) {
				t.Errorf("got %v, want code %q", err, tt.wantCode)
			}
			if !q.IsDefault() {
				t.Error("expected default QR code on error")
			}
		})
	}
}

func TestDefault(t *testing.T) {
	q := qrcode.Default()

	if !q.IsDefault() {
		t.Error("Default must be default")
	}
	if q.Content() != "" || q.LibraryVersion() != "" || q.ErrorCorrection() != qrcode.ErrorCorrectionUnknown {
		t.Error("unexpected default values")
	}
	if qrcode.Default() == q {
		t.Error("Default must return a new instance each time")
	}
}

func TestIsDefaultOnNil(t *testing.T) {
	var q *qrcode.QRCode

	if !q.IsDefault() {
		t.Error("a nil QR code must be default")
	}
}

func TestRestoreKeepsEverything(t *testing.T) {
	p := qrcode.RestoreParams{
		LotID:           uuid.New(),
		Content:         "https://rutadeorigen.co/l/x",
		ErrorCorrection: qrcode.ErrorCorrectionHigh,
		LibraryVersion:  "v2",
		CreatedAt:       time.Now().UTC(),
	}

	q := qrcode.Restore(p)

	if q.LotID() != p.LotID || q.Content() != p.Content || q.ErrorCorrection() != p.ErrorCorrection {
		t.Error("fields were not restored")
	}
	if q.LibraryVersion() != p.LibraryVersion || !q.CreatedAt().Equal(p.CreatedAt) {
		t.Error("fields were not restored")
	}
}
