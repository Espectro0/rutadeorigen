package qrcode

import (
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"

	"rutadeorigen/qr/internal/platform/exception"
	"rutadeorigen/qr/internal/platform/helper"
)

const ScanPathPrefix = "/l/"

type QRCode struct {
	lotID           uuid.UUID
	content         string
	errorCorrection ErrorCorrection
	libraryVersion  string
	createdAt       time.Time
}

var defaultQRCode = QRCode{
	lotID:           helper.DefaultUUID(),
	content:         helper.EmptyString,
	errorCorrection: ErrorCorrectionUnknown,
	libraryVersion:  helper.EmptyString,
	createdAt:       helper.DefaultDate(),
}

func Default() *QRCode {
	q := defaultQRCode
	return &q
}

func New(lotID uuid.UUID, publicBaseURL string, errorCorrection ErrorCorrection, libraryVersion string) (*QRCode, error) {
	libraryVersion = helper.ApplyTrim(libraryVersion)

	if helper.IsDefaultUUID(lotID) {
		return Default(), exception.New(exception.CodeLotViewInvalidID)
	}
	content, ok := buildContent(publicBaseURL, lotID)
	if !ok || helper.IsEmpty(libraryVersion) {
		return Default(), exception.New(exception.CodeQRCodeInvalidContent)
	}
	if !errorCorrection.IsValid() {
		return Default(), exception.New(exception.CodeQRCodeInvalidErrorCorrection)
	}

	return &QRCode{
		lotID:           lotID,
		content:         content,
		errorCorrection: errorCorrection,
		libraryVersion:  libraryVersion,
		createdAt:       time.Now().UTC(),
	}, nil
}

func buildContent(publicBaseURL string, lotID uuid.UUID) (string, bool) {
	parsed, err := url.Parse(helper.ApplyTrim(publicBaseURL))
	if err != nil || parsed.Host == helper.EmptyString {
		return helper.EmptyString, false
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return helper.EmptyString, false
	}

	base := parsed.Scheme + "://" + parsed.Host + strings.TrimRight(parsed.Path, "/")
	return base + ScanPathPrefix + lotID.String(), true
}

type RestoreParams struct {
	LotID           uuid.UUID
	Content         string
	ErrorCorrection ErrorCorrection
	LibraryVersion  string
	CreatedAt       time.Time
}

func Restore(p RestoreParams) *QRCode {
	return &QRCode{
		lotID:           p.LotID,
		content:         p.Content,
		errorCorrection: p.ErrorCorrection,
		libraryVersion:  p.LibraryVersion,
		createdAt:       p.CreatedAt,
	}
}

func (q *QRCode) LotID() uuid.UUID                 { return q.lotID }
func (q *QRCode) Content() string                  { return q.content }
func (q *QRCode) ErrorCorrection() ErrorCorrection { return q.errorCorrection }
func (q *QRCode) LibraryVersion() string           { return q.libraryVersion }
func (q *QRCode) CreatedAt() time.Time             { return q.createdAt }

func (q *QRCode) IsDefault() bool {
	return q == nil || helper.IsDefaultUUID(q.lotID)
}
