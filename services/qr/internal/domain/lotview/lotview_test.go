package lotview_test

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"rutadeorigen/qr/internal/domain/lotview"
	"rutadeorigen/qr/internal/platform/exception"
	"rutadeorigen/qr/internal/platform/helper"
)

func validParams() lotview.NewParams {
	return lotview.NewParams{
		LotID:        uuid.New(),
		Code:         "LT-001",
		Variety:      "Castillo",
		Process:      lotview.ProcessWashed,
		HarvestDate:  helper.DateUTC(time.Now().AddDate(0, -1, 0)),
		FarmID:       uuid.New(),
		FarmName:     "La Esperanza",
		FarmAltitude: 1800,
		ProducerID:   uuid.New(),
		ProducerName: "Maria Lopez",
		CustodianID:  uuid.New(),
		CurrentStage: lotview.StageNone,
		Status:       lotview.StatusActive,
		Version:      1,
		CreatedAt:    time.Now().Add(-time.Hour),
	}
}

func newPackaged(t *testing.T, status lotview.Status) *lotview.LotView {
	t.Helper()
	p := validParams()
	p.CurrentStage = lotview.StagePackaging
	p.Status = status
	v, err := lotview.New(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return v
}

func TestNewValid(t *testing.T) {
	p := validParams()
	p.Code = "  LT-001  "
	p.Variety = "  Castillo  "
	p.FarmName = "  La Esperanza  "
	p.ProducerName = "  Maria Lopez  "
	p.VoidReason = "  "

	v, err := lotview.New(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if v.LotID() != p.LotID || v.CustodianID() != p.CustodianID {
		t.Error("unexpected lot or custodian ID")
	}
	if v.Code() != "LT-001" || v.Variety() != "Castillo" {
		t.Errorf("expected trimmed values, got %q and %q", v.Code(), v.Variety())
	}
	if v.FarmID() != p.FarmID || v.FarmName() != "La Esperanza" || v.FarmAltitude() != 1800 {
		t.Error("unexpected farm snapshot")
	}
	if v.ProducerID() != p.ProducerID || v.ProducerName() != "Maria Lopez" {
		t.Error("unexpected producer snapshot")
	}
	if v.Process() != lotview.ProcessWashed || !v.HarvestDate().Equal(p.HarvestDate) {
		t.Error("unexpected process or harvest date")
	}
	if v.CurrentStage() != lotview.StageNone || v.Status() != lotview.StatusActive {
		t.Error("unexpected stage or status")
	}
	if v.VoidReason() != "" || v.Version() != 1 {
		t.Errorf("unexpected void reason %q or version %d", v.VoidReason(), v.Version())
	}
	if !v.CreatedAt().Equal(p.CreatedAt.UTC()) || v.UpdatedAt().IsZero() {
		t.Error("unexpected timestamps")
	}
	if v.IsDefault() {
		t.Error("a valid lot view must not be default")
	}
}

func TestNewWithoutCreatedAtUsesNow(t *testing.T) {
	p := validParams()
	p.CreatedAt = time.Time{}

	v, err := lotview.New(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v.CreatedAt().IsZero() {
		t.Error("expected a creation date")
	}
}

func TestNewInvalid(t *testing.T) {
	tests := []struct {
		name     string
		mutate   func(p *lotview.NewParams)
		wantCode exception.Code
	}{
		{"default lot ID", func(p *lotview.NewParams) { p.LotID = uuid.Nil }, exception.CodeLotViewInvalidID},
		{"code too short", func(p *lotview.NewParams) { p.Code = "LT" }, exception.CodeLotViewInvalidCode},
		{"code too long", func(p *lotview.NewParams) { p.Code = "LT-0000000000000000000000000001" }, exception.CodeLotViewInvalidCode},
		{"empty variety", func(p *lotview.NewParams) { p.Variety = "   " }, exception.CodeLotViewInvalidVariety},
		{"unknown process", func(p *lotview.NewParams) { p.Process = lotview.ProcessUnknown }, exception.CodeLotViewInvalidProcess},
		{"default harvest date", func(p *lotview.NewParams) { p.HarvestDate = time.Time{} }, exception.CodeLotViewInvalidHarvest},
		{"default farm ID", func(p *lotview.NewParams) { p.FarmID = uuid.Nil }, exception.CodeLotViewInvalidFarm},
		{"empty farm name", func(p *lotview.NewParams) { p.FarmName = " " }, exception.CodeLotViewInvalidFarm},
		{"negative altitude", func(p *lotview.NewParams) { p.FarmAltitude = -1 }, exception.CodeLotViewInvalidFarm},
		{"altitude too high", func(p *lotview.NewParams) { p.FarmAltitude = 3001 }, exception.CodeLotViewInvalidFarm},
		{"default producer ID", func(p *lotview.NewParams) { p.ProducerID = uuid.Nil }, exception.CodeLotViewInvalidProducer},
		{"empty producer name", func(p *lotview.NewParams) { p.ProducerName = "" }, exception.CodeLotViewInvalidProducer},
		{"default custodian ID", func(p *lotview.NewParams) { p.CustodianID = uuid.Nil }, exception.CodeLotViewInvalidCustodian},
		{"unknown stage", func(p *lotview.NewParams) { p.CurrentStage = lotview.StageUnknown }, exception.CodeLotViewInvalidStage},
		{"unknown status", func(p *lotview.NewParams) { p.Status = lotview.StatusUnknown }, exception.CodeLotViewInvalidStatus},
		{"zero version", func(p *lotview.NewParams) { p.Version = 0 }, exception.CodeLotViewInvalidVersion},
		{"negative version", func(p *lotview.NewParams) { p.Version = -2 }, exception.CodeLotViewInvalidVersion},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := validParams()
			tt.mutate(&p)

			v, err := lotview.New(p)
			if !exception.HasCode(err, tt.wantCode) {
				t.Errorf("got %v, want code %q", err, tt.wantCode)
			}
			if !v.IsDefault() {
				t.Error("expected default lot view on error")
			}
		})
	}
}

func TestNewInvalidCodeCarriesLengthLimits(t *testing.T) {
	const (
		wantMin = 3
		wantMax = 30
	)

	for _, code := range []string{"LT", "LT-0000000000000000000000000001"} {
		p := validParams()
		p.Code = code

		_, err := lotview.New(p)

		ex, ok := exception.From(err)
		if !ok {
			t.Fatalf("code %q: expected an exception, got %v", code, err)
		}
		if ex.Params()["min"] != wantMin || ex.Params()["max"] != wantMax {
			t.Errorf("code %q: params = %v, want min=%d max=%d", code, ex.Params(), wantMin, wantMax)
		}
	}
}

func TestDefault(t *testing.T) {
	v := lotview.Default()

	if !v.IsDefault() {
		t.Error("Default must be default")
	}
	if v.Process() != lotview.ProcessUnknown || v.Status() != lotview.StatusUnknown || v.CurrentStage() != lotview.StageNone {
		t.Error("unexpected default enums")
	}
	if v.Code() != "" || v.Version() != 0 || v.IsActive() || v.IsPackaged() {
		t.Error("unexpected default values")
	}
	if lotview.Default() == v {
		t.Error("Default must return a new instance each time")
	}
}

func TestIsDefaultOnNil(t *testing.T) {
	var v *lotview.LotView

	if !v.IsDefault() {
		t.Error("a nil lot view must be default")
	}
}

func TestRestoreKeepsEverything(t *testing.T) {
	now := time.Now().UTC()
	p := lotview.RestoreParams{
		LotID:        uuid.New(),
		Code:         "LT-009",
		Variety:      "Caturra",
		Process:      lotview.ProcessHoney,
		HarvestDate:  now.AddDate(0, -2, 0),
		FarmID:       uuid.New(),
		FarmName:     "El Mirador",
		FarmAltitude: 1500,
		ProducerID:   uuid.New(),
		ProducerName: "Pedro Ruiz",
		CustodianID:  uuid.New(),
		CurrentStage: lotview.StageRoasting,
		Status:       lotview.StatusVoided,
		VoidReason:   "wrong batch",
		Version:      4,
		CreatedAt:    now.Add(-48 * time.Hour),
		UpdatedAt:    now,
	}

	v := lotview.Restore(p)

	if v.LotID() != p.LotID || v.Code() != p.Code || v.Variety() != p.Variety || v.Process() != p.Process {
		t.Error("lot fields were not restored")
	}
	if v.FarmID() != p.FarmID || v.FarmName() != p.FarmName || v.FarmAltitude() != p.FarmAltitude {
		t.Error("farm fields were not restored")
	}
	if v.ProducerID() != p.ProducerID || v.ProducerName() != p.ProducerName || v.CustodianID() != p.CustodianID {
		t.Error("producer or custodian fields were not restored")
	}
	if v.CurrentStage() != p.CurrentStage || v.Status() != p.Status || v.VoidReason() != p.VoidReason || v.Version() != p.Version {
		t.Error("state fields were not restored")
	}
	if !v.HarvestDate().Equal(p.HarvestDate) || !v.CreatedAt().Equal(p.CreatedAt) || !v.UpdatedAt().Equal(p.UpdatedAt) {
		t.Error("dates were not restored")
	}
}

func TestIsNewerThan(t *testing.T) {
	older := lotview.Restore(lotview.RestoreParams{LotID: uuid.New(), Version: 2})
	newer := lotview.Restore(lotview.RestoreParams{LotID: uuid.New(), Version: 3})
	same := lotview.Restore(lotview.RestoreParams{LotID: uuid.New(), Version: 2})

	if !newer.IsNewerThan(older) {
		t.Error("version 3 must be newer than version 2")
	}
	if older.IsNewerThan(newer) {
		t.Error("version 2 must not be newer than version 3")
	}
	if same.IsNewerThan(older) {
		t.Error("the same version must not be newer")
	}
	if !older.IsNewerThan(lotview.Default()) {
		t.Error("any lot view must be newer than the default one")
	}
	if !older.IsNewerThan(nil) {
		t.Error("any lot view must be newer than nil")
	}
}

func TestCanGenerateImage(t *testing.T) {
	t.Run("active and packaged", func(t *testing.T) {
		if err := newPackaged(t, lotview.StatusActive).CanGenerateImage(); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("voided and packaged", func(t *testing.T) {
		err := newPackaged(t, lotview.StatusVoided).CanGenerateImage()
		if !exception.HasCode(err, exception.CodeLotViewImageNotAvailable) {
			t.Errorf("got %v, want image not available", err)
		}
	})

	for _, stage := range []lotview.Stage{lotview.StageNone, lotview.StageFarm, lotview.StageRoasting} {
		t.Run(stage.String(), func(t *testing.T) {
			p := validParams()
			p.CurrentStage = stage
			v, err := lotview.New(p)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			err = v.CanGenerateImage()
			if !exception.HasCode(err, exception.CodeLotViewImageNotAvailable) {
				t.Errorf("got %v, want image not available", err)
			}
		})
	}

	t.Run("default lot view", func(t *testing.T) {
		err := lotview.Default().CanGenerateImage()
		if !exception.HasCode(err, exception.CodeLotViewImageNotAvailable) {
			t.Errorf("got %v, want image not available", err)
		}
	})
}
