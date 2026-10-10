package lot_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"rutadeorigen/traceability/internal/domain/lot"
)

func validHarvestDate() time.Time {
	return time.Now().AddDate(0, -1, 0)
}

func newValidLot(t *testing.T) *lot.Lot {
	t.Helper()
	l, err := lot.New(uuid.New(), uuid.New(), "LT-001", "Castillo", lot.ProcessWashed, validHarvestDate())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return l
}

func TestNewValid(t *testing.T) {
	farmID, custodianID := uuid.New(), uuid.New()
	harvestDate := validHarvestDate()

	l, err := lot.New(farmID, custodianID, "  LT-001  ", "  Castillo  ", lot.ProcessWashed, harvestDate)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if l.Code() != "LT-001" || l.Variety() != "Castillo" {
		t.Errorf("expected trimmed values, got %q and %q", l.Code(), l.Variety())
	}
	if l.FarmID() != farmID || l.CustodianID() != custodianID {
		t.Error("unexpected farm or custodian")
	}
	if l.Process() != lot.ProcessWashed || !l.HarvestDate().Equal(harvestDate) {
		t.Error("unexpected process or harvest date")
	}
	if l.ID() == uuid.Nil || l.CreatedAt().IsZero() {
		t.Error("expected generated ID and creation date")
	}
}

func TestNewInitialState(t *testing.T) {
	l := newValidLot(t)

	if !l.IsActive() || l.Status() != lot.StatusActive {
		t.Error("a new lot must be active")
	}
	if l.HasEvents() || l.CurrentStage() != lot.StageNone {
		t.Error("a new lot must not have events")
	}
	if l.VoidReason() != "" {
		t.Errorf("expected empty void reason, got %q", l.VoidReason())
	}
	if l.Version() != 1 {
		t.Errorf("expected version 1, got %d", l.Version())
	}
}

func TestNewInvalid(t *testing.T) {
	tests := []struct {
		name        string
		farmID      uuid.UUID
		custodianID uuid.UUID
		code        string
		variety     string
		process     lot.Process
		harvestDate time.Time
		want        error
	}{
		{"default farm ID", uuid.Nil, uuid.New(), "LT-001", "Castillo", lot.ProcessWashed, validHarvestDate(), lot.ErrInvalidFarm},
		{"default custodian ID", uuid.New(), uuid.Nil, "LT-001", "Castillo", lot.ProcessWashed, validHarvestDate(), lot.ErrInvalidCustodian},
		{"code too short", uuid.New(), uuid.New(), "LT", "Castillo", lot.ProcessWashed, validHarvestDate(), lot.ErrInvalidCode},
		{"code too long", uuid.New(), uuid.New(), "LT-0000000000000000000000000001", "Castillo", lot.ProcessWashed, validHarvestDate(), lot.ErrInvalidCode},
		{"empty variety", uuid.New(), uuid.New(), "LT-001", "   ", lot.ProcessWashed, validHarvestDate(), lot.ErrInvalidVariety},
		{"unknown process", uuid.New(), uuid.New(), "LT-001", "Castillo", lot.ProcessUnknown, validHarvestDate(), lot.ErrInvalidProcess},
		{"default harvest date", uuid.New(), uuid.New(), "LT-001", "Castillo", lot.ProcessWashed, time.Time{}, lot.ErrInvalidHarvestDate},
		{"future harvest date", uuid.New(), uuid.New(), "LT-001", "Castillo", lot.ProcessWashed, time.Now().AddDate(0, 0, 1), lot.ErrInvalidHarvestDate},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l, err := lot.New(tt.farmID, tt.custodianID, tt.code, tt.variety, tt.process, tt.harvestDate)
			if !errors.Is(err, tt.want) {
				t.Errorf("got %v, want %v", err, tt.want)
			}
			if !l.IsDefault() {
				t.Error("expected default lot on error")
			}
		})
	}
}

func TestVoid(t *testing.T) {
	l := newValidLot(t)

	if err := l.Void("   "); !errors.Is(err, lot.ErrEmptyVoidReason) {
		t.Errorf("empty reason: got %v, want %v", err, lot.ErrEmptyVoidReason)
	}
	if l.Version() != 1 {
		t.Error("a failed void must not change the version")
	}

	if err := l.Void("  Registered on the wrong farm  "); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if l.IsActive() || l.Status() != lot.StatusVoided {
		t.Error("expected the lot to be voided")
	}
	if l.VoidReason() != "Registered on the wrong farm" {
		t.Errorf("expected trimmed reason, got %q", l.VoidReason())
	}
	if l.Version() != 2 {
		t.Errorf("expected version 2, got %d", l.Version())
	}

	if err := l.Void("Again"); !errors.Is(err, lot.ErrVoided) {
		t.Errorf("void twice: got %v, want %v", err, lot.ErrVoided)
	}
}

func TestCanBeDeleted(t *testing.T) {
	if err := newValidLot(t).CanBeDeleted(); err != nil {
		t.Errorf("a lot without events must be deletable, got %v", err)
	}

	withEvents := lot.Restore(lot.RestoreParams{
		ID:           uuid.New(),
		CurrentStage: lot.StageFarm,
		Status:       lot.StatusActive,
	})
	if err := withEvents.CanBeDeleted(); !errors.Is(err, lot.ErrHasEvents) {
		t.Errorf("got %v, want %v", err, lot.ErrHasEvents)
	}
}

func TestRestore(t *testing.T) {
	params := lot.RestoreParams{
		ID:           uuid.New(),
		FarmID:       uuid.New(),
		CustodianID:  uuid.New(),
		Code:         "LT-001",
		Variety:      "Castillo",
		Process:      lot.ProcessHoney,
		HarvestDate:  time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC),
		CurrentStage: lot.StageNone,
		Status:       lot.StatusVoided,
		VoidReason:   "Wrong farm",
		Version:      3,
		CreatedAt:    time.Date(2026, time.September, 2, 0, 0, 0, 0, time.UTC),
	}

	l := lot.Restore(params)

	if l.ID() != params.ID || l.Version() != 3 || l.VoidReason() != "Wrong farm" {
		t.Error("expected stored values to be kept")
	}
	if l.IsActive() {
		t.Error("expected the restored lot to be voided")
	}
}

func TestDefault(t *testing.T) {
	d := lot.Default()

	if !d.IsDefault() {
		t.Error("expected Default() to be the default lot")
	}
	if d.ID() != uuid.Nil || d.FarmID() != uuid.Nil || d.CustodianID() != uuid.Nil || d.Code() != "" {
		t.Error("expected default values")
	}
	if d.Status() != lot.StatusUnknown || d.Process() != lot.ProcessUnknown || d.Version() != 0 {
		t.Error("expected unknown status and process, and version 0")
	}
	if d.IsActive() {
		t.Error("the default lot must not be active")
	}
	if err := d.Void("Any reason"); !errors.Is(err, lot.ErrVoided) {
		t.Errorf("the default lot must not be voidable, got %v", err)
	}
	if d == lot.Default() {
		t.Error("expected each call to return a new copy")
	}
	if newValidLot(t).IsDefault() {
		t.Error("a new lot must not be the default lot")
	}

	var nilLot *lot.Lot
	if !nilLot.IsDefault() {
		t.Error("a nil lot must be treated as default")
	}
}
