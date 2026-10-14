package assembler_test

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"rutadeorigen/traceability/internal/infrastructure/output/repositories/structured/postgres/assembler"
	"rutadeorigen/traceability/internal/infrastructure/output/repositories/structured/postgres/entity"
)

// pgx returns TIMESTAMPTZ values in the process local zone; simulate a
// machine running in Bogotá to check that the domain always receives UTC.
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

func lotEntity() entity.Lot {
	return entity.Lot{
		ID:           uuid.New(),
		FarmID:       uuid.New(),
		CustodianID:  uuid.New(),
		Code:         "LT-001",
		Variety:      "Castillo",
		Process:      "washed",
		HarvestDate:  time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC),
		CurrentStage: "none",
		Status:       "active",
		VoidReason:   "",
		Version:      1,
		CreatedAt:    time.Date(2026, time.October, 13, 20, 0, 0, 0, bogota),
	}
}

func TestLotToDomainReturnsUTC(t *testing.T) {
	e := lotEntity()

	l := assembler.Lot{}.ToDomain(e)

	assertUTC(t, "created_at", l.CreatedAt(), e.CreatedAt)
	assertUTC(t, "harvest_date", l.HarvestDate(), e.HarvestDate)
}

func TestLotRoundTrip(t *testing.T) {
	e := lotEntity()

	got := assembler.Lot{}.ToEntity(assembler.Lot{}.ToDomain(e))

	if got.ID != e.ID || got.FarmID != e.FarmID || got.CustodianID != e.CustodianID ||
		got.Code != e.Code || got.Variety != e.Variety || got.Process != e.Process ||
		got.CurrentStage != e.CurrentStage || got.Status != e.Status ||
		got.VoidReason != e.VoidReason || got.Version != e.Version {
		t.Errorf("round trip changed the lot:\n got  %+v\n want %+v", got, e)
	}
	assertUTC(t, "created_at", got.CreatedAt, e.CreatedAt)
	assertUTC(t, "harvest_date", got.HarvestDate, e.HarvestDate)
}

func TestLotToEntityOfDefaultLot(t *testing.T) {
	got := assembler.Lot{}.ToEntity(assembler.Lot{}.ToDomain(entity.Lot{}))

	if got != (entity.Lot{}) {
		t.Errorf("default lot must map to an empty entity, got %+v", got)
	}
}

func TestActorToDomainReturnsUTC(t *testing.T) {
	e := entity.Actor{
		ID:        uuid.New(),
		UserID:    "user_123",
		Type:      "producer",
		Name:      "Productor",
		Status:    "active",
		CreatedAt: time.Date(2026, time.October, 13, 20, 0, 0, 0, bogota),
	}

	a := assembler.Actor{}.ToDomain(e)
	assertUTC(t, "created_at", a.CreatedAt(), e.CreatedAt)

	got := assembler.Actor{}.ToEntity(a)
	if got.ID != e.ID || got.UserID != e.UserID || got.Type != e.Type || got.Name != e.Name || got.Status != e.Status {
		t.Errorf("round trip changed the actor:\n got  %+v\n want %+v", got, e)
	}
	assertUTC(t, "created_at", got.CreatedAt, e.CreatedAt)
}

func TestFarmToDomainReturnsUTC(t *testing.T) {
	e := entity.Farm{
		ID:         uuid.New(),
		ProducerID: uuid.New(),
		Name:       "Finca",
		Altitude:   1800,
		CreatedAt:  time.Date(2026, time.October, 13, 20, 0, 0, 0, bogota),
	}

	f := assembler.Farm{}.ToDomain(e)
	assertUTC(t, "created_at", f.CreatedAt(), e.CreatedAt)

	got := assembler.Farm{}.ToEntity(f)
	if got.ID != e.ID || got.ProducerID != e.ProducerID || got.Name != e.Name || got.Altitude != e.Altitude {
		t.Errorf("round trip changed the farm:\n got  %+v\n want %+v", got, e)
	}
	assertUTC(t, "created_at", got.CreatedAt, e.CreatedAt)
}
