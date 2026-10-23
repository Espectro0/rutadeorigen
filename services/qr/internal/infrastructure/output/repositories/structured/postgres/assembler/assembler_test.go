package assembler_test

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"rutadeorigen/qr/internal/domain/lotview"
	"rutadeorigen/qr/internal/domain/qrcode"
	"rutadeorigen/qr/internal/domain/redirect"
	"rutadeorigen/qr/internal/infrastructure/output/repositories/structured/postgres/assembler"
	"rutadeorigen/qr/internal/infrastructure/output/repositories/structured/postgres/entity"
)

var bogota = time.FixedZone("COT", -5*60*60)

func assertUTC(t *testing.T, field string, got, want time.Time) {
	t.Helper()
	if !got.Equal(want) {
		t.Errorf("%s = %v, want the instant %v", field, got, want)
	}
	if got.Location() != time.UTC {
		t.Errorf("%s location = %v, want UTC", field, got.Location())
	}
}

func lotViewEntity() entity.LotView {
	return entity.LotView{
		LotID:        uuid.New(),
		Code:         "LT-001",
		Variety:      "Castillo",
		Process:      "washed",
		HarvestDate:  time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC),
		FarmID:       uuid.New(),
		FarmName:     "La Esperanza",
		FarmAltitude: 1800,
		ProducerID:   uuid.New(),
		ProducerName: "Maria Lopez",
		CustodianID:  uuid.New(),
		CurrentStage: "packaging",
		Status:       "voided",
		VoidReason:   "mislabeled",
		Version:      4,
		CreatedAt:    time.Date(2026, 9, 1, 10, 0, 0, 0, bogota),
		UpdatedAt:    time.Date(2026, 9, 2, 10, 0, 0, 0, bogota),
	}
}

func TestLotViewRoundTrip(t *testing.T) {
	a := assembler.LotView{}
	e := lotViewEntity()

	view := a.ToDomain(e)

	if view.LotID() != e.LotID || view.Process() != lotview.ProcessWashed ||
		view.CurrentStage() != lotview.StagePackaging || view.Status() != lotview.StatusVoided ||
		view.VoidReason() != "mislabeled" || view.Version() != 4 || view.FarmAltitude() != 1800 {
		t.Errorf("unexpected domain %+v", view)
	}
	assertUTC(t, "created_at", view.CreatedAt(), e.CreatedAt)
	assertUTC(t, "updated_at", view.UpdatedAt(), e.UpdatedAt)

	back := a.ToEntity(view)
	if back.LotID != e.LotID || back.Process != "washed" || back.CurrentStage != "packaging" ||
		back.Status != "voided" || back.Version != 4 || back.ProducerName != "Maria Lopez" {
		t.Errorf("unexpected entity %+v", back)
	}
	assertUTC(t, "entity created_at", back.CreatedAt, e.CreatedAt)
}

func TestLotViewDefaultMapsToZeroEntity(t *testing.T) {
	if got := (assembler.LotView{}).ToEntity(lotview.Default()); got != (entity.LotView{}) {
		t.Errorf("expected zero entity, got %+v", got)
	}
}

func TestQRCodeRoundTrip(t *testing.T) {
	a := assembler.QRCode{}
	e := entity.QRCode{
		LotID:           uuid.New(),
		Content:         "https://example.com/l/x",
		ErrorCorrection: "H",
		LibraryVersion:  "v1",
		CreatedAt:       time.Date(2026, 9, 1, 10, 0, 0, 0, bogota),
	}

	code := a.ToDomain(e)

	if code.LotID() != e.LotID || code.ErrorCorrection() != qrcode.ErrorCorrectionHigh || code.Content() != e.Content {
		t.Errorf("unexpected domain %+v", code)
	}
	assertUTC(t, "created_at", code.CreatedAt(), e.CreatedAt)

	back := a.ToEntity(code)
	if back.LotID != e.LotID || back.ErrorCorrection != "H" || back.LibraryVersion != "v1" {
		t.Errorf("unexpected entity %+v", back)
	}
	if got := a.ToEntity(qrcode.Default()); got != (entity.QRCode{}) {
		t.Errorf("expected zero entity, got %+v", got)
	}
}

func TestRedirectRoundTrip(t *testing.T) {
	a := assembler.Redirect{}
	e := entity.Redirect{
		ID:             uuid.New(),
		SourcePrefix:   "/l/",
		TargetTemplate: "/lotes/{id}",
		StatusCode:     301,
		Active:         true,
		CreatedAt:      time.Date(2026, 9, 1, 10, 0, 0, 0, bogota),
	}

	rule := a.ToDomain(e)

	if rule.ID() != e.ID || rule.SourcePrefix() != "/l/" || rule.StatusCode() != 301 || !rule.Active() {
		t.Errorf("unexpected domain %+v", rule)
	}
	assertUTC(t, "created_at", rule.CreatedAt(), e.CreatedAt)

	back := a.ToEntity(rule)
	if back.ID != e.ID || back.TargetTemplate != "/lotes/{id}" || !back.Active {
		t.Errorf("unexpected entity %+v", back)
	}
	if got := a.ToEntity(redirect.Default()); got != (entity.Redirect{}) {
		t.Errorf("expected zero entity, got %+v", got)
	}
}
