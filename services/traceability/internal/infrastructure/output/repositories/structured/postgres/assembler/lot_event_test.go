package assembler_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"

	"rutadeorigen/traceability/internal/domain/lot"
	"rutadeorigen/traceability/internal/infrastructure/output/repositories/structured/postgres/assembler"
	"rutadeorigen/traceability/internal/infrastructure/output/repositories/structured/postgres/entity"
	"rutadeorigen/traceability/internal/platform/exception"
)

func farmDetails() lot.FarmDetails {
	return lot.FarmDetails{
		DryingMethod:      lot.DryingMethodAfricanBed,
		DryingDays:        15,
		MoisturePercent:   11.5,
		ParchmentWeightKg: 120.5,
	}
}

func roastingDetails() lot.RoastingDetails {
	return lot.RoastingDetails{
		RoastLevel:      lot.RoastLevelMedium,
		DurationMinutes: 14,
		GreenWeightKg:   100,
		RoastedWeightKg: 84.25,
	}
}

func packagingDetails() lot.PackagingDetails {
	return lot.PackagingDetails{
		Grind:          lot.GrindWholeBean,
		PackageWeightG: 250,
		Units:          40,
		BestBefore:     time.Now().AddDate(0, 6, 0).Format(time.DateOnly),
	}
}

func newStageEvent(t *testing.T, previous *lot.Event, lotID uuid.UUID, stage lot.Stage, details lot.StageDetails) *lot.Event {
	t.Helper()

	e, err := lot.NewStageEvent(lot.StageEventParams{
		LotID:       lotID,
		HarvestDate: time.Now().AddDate(0, -1, 0),
		Previous:    previous,
		Stage:       stage,
		EventDate:   time.Now().AddDate(0, 0, -1),
		Note:        "Dried on raised beds",
		ActorID:     uuid.New(),
		ActorRole:   "producer",
		Details:     details,
	})
	if err != nil {
		t.Fatal(err)
	}
	return e
}

func lotEventEntity(t *testing.T) entity.LotEvent {
	t.Helper()

	details, err := json.Marshal(farmDetails())
	if err != nil {
		t.Fatal(err)
	}

	return entity.LotEvent{
		ID:           uuid.New(),
		LotID:        uuid.New(),
		Sequence:     2,
		Type:         "stage",
		Stage:        "farm",
		EventDate:    time.Date(2026, time.October, 1, 0, 0, 0, 0, time.UTC),
		Note:         "Dried on raised beds",
		ActorID:      uuid.New(),
		ActorRole:    "producer",
		Details:      details,
		PreviousHash: "previous",
		Hash:         "current",
		RegisteredAt: time.Date(2026, time.October, 2, 15, 4, 5, 123456000, bogota),
	}
}

func normalizeLikeJSONB(t *testing.T, raw []byte) []byte {
	t.Helper()

	var generic map[string]any
	if err := json.Unmarshal(raw, &generic); err != nil {
		t.Fatal(err)
	}
	normalized, err := json.MarshalIndent(generic, "", "   ")
	if err != nil {
		t.Fatal(err)
	}
	return normalized
}

func TestLotEventToDomain(t *testing.T) {
	t.Run("maps every field", func(t *testing.T) {
		e := lotEventEntity(t)

		got, err := assembler.LotEvent{}.ToDomain(e)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if got.ID() != e.ID || got.LotID() != e.LotID || got.Sequence() != 2 {
			t.Error("unexpected identity")
		}
		if got.Type() != lot.EventTypeStage || got.Stage() != lot.StageFarm {
			t.Errorf("unexpected classification: %q / %q", got.Type(), got.Stage())
		}
		if got.Note() != e.Note || got.ActorID() != e.ActorID || got.ActorRole() != "producer" {
			t.Error("unexpected note or actor")
		}
		if got.PreviousHash() != "previous" || got.Hash() != "current" {
			t.Error("unexpected hashes")
		}
		if got.Details() != lot.StageDetails(farmDetails()) {
			t.Errorf("Details: got %+v, want %+v", got.Details(), farmDetails())
		}
	})

	t.Run("returns UTC times", func(t *testing.T) {
		e := lotEventEntity(t)
		e.EventDate = time.Date(2026, time.October, 1, 0, 0, 0, 0, bogota)

		got, err := assembler.LotEvent{}.ToDomain(e)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		assertUTC(t, "event_date", got.EventDate(), e.EventDate)
		assertUTC(t, "registered_at", got.RegisteredAt(), e.RegisteredAt)
	})

	t.Run("decodes the details of every stage into their own type", func(t *testing.T) {
		tests := []struct {
			stage   string
			details lot.StageDetails
		}{
			{"farm", farmDetails()},
			{"roasting", roastingDetails()},
			{"packaging", packagingDetails()},
		}

		for _, tt := range tests {
			t.Run(tt.stage, func(t *testing.T) {
				e := lotEventEntity(t)
				e.Stage = tt.stage
				raw, err := json.Marshal(tt.details)
				if err != nil {
					t.Fatal(err)
				}
				e.Details = raw

				got, err := assembler.LotEvent{}.ToDomain(e)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				if got.Details() != tt.details {
					t.Errorf("got %+v, want %+v", got.Details(), tt.details)
				}
			})
		}
	})
}

func TestLotEventToDomainFailures(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*entity.LotEvent)
	}{
		{"empty details", func(e *entity.LotEvent) { e.Details = nil }},
		{"null details", func(e *entity.LotEvent) { e.Details = []byte("null") }},
		{"invalid JSON", func(e *entity.LotEvent) { e.Details = []byte(`{"drying_days":`) }},
		{"details as an array", func(e *entity.LotEvent) { e.Details = []byte(`[]`) }},
		{"unknown field", func(e *entity.LotEvent) { e.Details = []byte(`{"drying_method":"sun_patio","altitude":1500}`) }},
		{"wrong type", func(e *entity.LotEvent) { e.Details = []byte(`{"drying_days":"ten"}`) }},
		{"trailing data", func(e *entity.LotEvent) { e.Details = append(e.Details, []byte(` {}`)...) }},
		{"details of another stage", func(e *entity.LotEvent) { e.Stage = "roasting" }},
		{"stage none", func(e *entity.LotEvent) { e.Stage = "none" }},
		{"unknown stage", func(e *entity.LotEvent) { e.Stage = "export" }},
		{"empty stage", func(e *entity.LotEvent) { e.Stage = "" }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := lotEventEntity(t)
			tt.mutate(&e)

			got, err := assembler.LotEvent{}.ToDomain(e)

			ex, ok := exception.From(err)
			if !ok || ex.Kind() != exception.KindInternal {
				t.Errorf("got %v, want an internal error", err)
			}
			if !got.IsDefault() {
				t.Error("expected the default event on error")
			}
		})
	}
}

func TestLotEventToEntity(t *testing.T) {
	t.Run("maps every field", func(t *testing.T) {
		event := newStageEvent(t, nil, uuid.New(), lot.StageFarm, farmDetails())

		got, err := assembler.LotEvent{}.ToEntity(event)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if got.ID != event.ID() || got.LotID != event.LotID() || got.Sequence != 1 {
			t.Error("unexpected identity")
		}
		if got.Type != "stage" || got.Stage != "farm" {
			t.Errorf("unexpected classification: %q / %q", got.Type, got.Stage)
		}
		if got.Note != event.Note() || got.ActorID != event.ActorID() || got.ActorRole != "producer" {
			t.Error("unexpected note or actor")
		}
		if got.PreviousHash != lot.GenesisHash || got.Hash != event.Hash() {
			t.Error("unexpected hashes")
		}
		assertUTC(t, "event_date", got.EventDate, event.EventDate())
		assertUTC(t, "registered_at", got.RegisteredAt, event.RegisteredAt())
	})

	t.Run("serializes the details with the keys of the hash", func(t *testing.T) {
		event := newStageEvent(t, nil, uuid.New(), lot.StageFarm, farmDetails())

		got, err := assembler.LotEvent{}.ToEntity(event)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		want := `{"drying_method":"african_bed","drying_days":15,"moisture_percent":11.5,"parchment_weight_kg":120.5}`
		if string(got.Details) != want {
			t.Errorf("Details: got %s, want %s", got.Details, want)
		}
	})

	t.Run("a default or nil event maps to an empty entity", func(t *testing.T) {
		for _, event := range []*lot.Event{lot.DefaultEvent(), nil} {
			got, err := assembler.LotEvent{}.ToEntity(event)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.ID != uuid.Nil || got.Details != nil || got.Hash != "" {
				t.Errorf("expected an empty entity, got %+v", got)
			}
		}
	})

	t.Run("an event without details cannot be stored", func(t *testing.T) {
		event := lot.RestoreEvent(lot.RestoreEventParams{
			ID:        uuid.New(),
			LotID:     uuid.New(),
			Sequence:  1,
			EventType: lot.EventTypeStage,
			Stage:     lot.StageFarm,
		})

		_, err := assembler.LotEvent{}.ToEntity(event)

		ex, ok := exception.From(err)
		if !ok || ex.Kind() != exception.KindInternal {
			t.Errorf("got %v, want an internal error", err)
		}
	})
}

func TestLotEventRoundTripKeepsTheHashValid(t *testing.T) {
	lotID := uuid.New()
	farm := newStageEvent(t, nil, lotID, lot.StageFarm, farmDetails())
	roasting := newStageEvent(t, farm, lotID, lot.StageRoasting, roastingDetails())
	packaging := newStageEvent(t, roasting, lotID, lot.StagePackaging, packagingDetails())

	for _, event := range []*lot.Event{farm, roasting, packaging} {
		t.Run(event.Stage().String(), func(t *testing.T) {
			stored, err := assembler.LotEvent{}.ToEntity(event)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			stored.Details = normalizeLikeJSONB(t, stored.Details)
			stored.RegisteredAt = stored.RegisteredAt.In(bogota)

			loaded, err := assembler.LotEvent{}.ToDomain(stored)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !loaded.VerifyHash() {
				t.Error("the hash must verify after PostgreSQL normalizes the JSON")
			}
			if loaded.Hash() != event.Hash() || loaded.Details() != event.Details() {
				t.Error("unexpected content after the round trip")
			}
		})
	}
}

func TestLotEventTamperedDetailsFailTheHash(t *testing.T) {
	event := newStageEvent(t, nil, uuid.New(), lot.StageFarm, farmDetails())
	stored, err := assembler.LotEvent{}.ToEntity(event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tampered := farmDetails()
	tampered.DryingDays++
	stored.Details, err = json.Marshal(tampered)
	if err != nil {
		t.Fatal(err)
	}

	loaded, err := assembler.LotEvent{}.ToDomain(stored)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if loaded.VerifyHash() {
		t.Error("changed details must not verify")
	}
}
