package farm_test

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"rutadeorigen/traceability/internal/domain/farm"
	"rutadeorigen/traceability/internal/platform/exception"
)

func TestNewValid(t *testing.T) {
	producerID := uuid.New()

	f, err := farm.New(producerID, "  La Esperanza  ", 1650)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.Name() != "La Esperanza" {
		t.Errorf("expected trimmed name, got %q", f.Name())
	}
	if f.ProducerID() != producerID || f.Altitude() != 1650 {
		t.Error("unexpected producer or altitude")
	}
	if f.ID() == uuid.Nil || f.CreatedAt().IsZero() {
		t.Error("expected generated ID and creation date")
	}
}

func TestNewInvalid(t *testing.T) {
	tests := []struct {
		name       string
		producerID uuid.UUID
		farmName   string
		altitude   int
		wantCode   exception.Code
	}{
		{"default producer ID", uuid.Nil, "La Esperanza", 1650, exception.CodeFarmInvalidProducer},
		{"name too short", uuid.New(), "LE", 1650, exception.CodeFarmInvalidName},
		{"altitude below range", uuid.New(), "La Esperanza", -1, exception.CodeFarmInvalidAltitude},
		{"altitude above range", uuid.New(), "La Esperanza", 3001, exception.CodeFarmInvalidAltitude},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := farm.New(tt.producerID, tt.farmName, tt.altitude)
			if !exception.HasCode(err, tt.wantCode) {
				t.Errorf("got %v, want code %q", err, tt.wantCode)
			}
			if !f.IsDefault() {
				t.Error("expected default farm on error")
			}
		})
	}
}

func TestNewInvalidCarriesLimits(t *testing.T) {
	tests := []struct {
		name     string
		farmName string
		altitude int
		wantMin  int
		wantMax  int
	}{
		{"name", "LE", 1650, 3, 100},
		{"altitude", "La Esperanza", -1, 0, 3000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := farm.New(uuid.New(), tt.farmName, tt.altitude)

			ex, ok := exception.From(err)
			if !ok {
				t.Fatalf("expected an exception, got %v", err)
			}
			if ex.Params()["min"] != tt.wantMin || ex.Params()["max"] != tt.wantMax {
				t.Errorf("params = %v, want min=%d max=%d", ex.Params(), tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestNewAltitudeLimits(t *testing.T) {
	for _, altitude := range []int{0, 3000} {
		if _, err := farm.New(uuid.New(), "La Esperanza", altitude); err != nil {
			t.Errorf("altitude %d: unexpected error %v", altitude, err)
		}
	}
}

func TestBelongsTo(t *testing.T) {
	producerID := uuid.New()
	f, _ := farm.New(producerID, "La Esperanza", 1650)

	if !f.BelongsTo(producerID) {
		t.Error("expected the farm to belong to its producer")
	}
	if f.BelongsTo(uuid.New()) {
		t.Error("expected the farm not to belong to another actor")
	}
	if f.BelongsTo(uuid.Nil) {
		t.Error("expected the farm not to belong to the default UUID")
	}
}

func TestRestore(t *testing.T) {
	id, producerID := uuid.New(), uuid.New()
	createdAt := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

	f := farm.Restore(id, producerID, "La Esperanza", 1650, createdAt)

	if f.ID() != id || f.ProducerID() != producerID || !f.CreatedAt().Equal(createdAt) {
		t.Error("expected stored values to be kept")
	}
}

func TestDefault(t *testing.T) {
	d := farm.Default()

	if !d.IsDefault() {
		t.Error("expected Default() to be the default farm")
	}
	if d.ID() != uuid.Nil || d.ProducerID() != uuid.Nil || d.Name() != "" || d.Altitude() != 0 {
		t.Error("expected default values")
	}
	if d.BelongsTo(uuid.Nil) || d.BelongsTo(uuid.New()) {
		t.Error("the default farm must not belong to anyone")
	}
	if d == farm.Default() {
		t.Error("expected each call to return a new copy")
	}

	created, _ := farm.New(uuid.New(), "La Esperanza", 1650)
	if created.IsDefault() {
		t.Error("a new farm must not be the default farm")
	}

	var nilFarm *farm.Farm
	if !nilFarm.IsDefault() {
		t.Error("a nil farm must be treated as default")
	}
}
