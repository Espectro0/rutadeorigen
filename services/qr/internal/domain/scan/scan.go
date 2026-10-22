package scan

import (
	"time"

	"github.com/google/uuid"

	"rutadeorigen/qr/internal/platform/exception"
	"rutadeorigen/qr/internal/platform/helper"
)

type Scan struct {
	lotID     uuid.UUID
	scannedAt time.Time
}

var defaultScan = Scan{
	lotID:     helper.DefaultUUID(),
	scannedAt: helper.DefaultDate(),
}

func Default() *Scan {
	s := defaultScan
	return &s
}

func New(lotID uuid.UUID) (*Scan, error) {
	if helper.IsDefaultUUID(lotID) {
		return Default(), exception.New(exception.CodeLotViewInvalidID)
	}

	return &Scan{
		lotID:     lotID,
		scannedAt: time.Now().UTC(),
	}, nil
}

func Restore(lotID uuid.UUID, scannedAt time.Time) *Scan {
	return &Scan{
		lotID:     lotID,
		scannedAt: scannedAt,
	}
}

func (s *Scan) LotID() uuid.UUID     { return s.lotID }
func (s *Scan) ScannedAt() time.Time { return s.scannedAt }

func (s *Scan) IsDefault() bool {
	return s == nil || helper.IsDefaultUUID(s.lotID)
}
