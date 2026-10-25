package lotmapper_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"

	"rutadeorigen/traceability/internal/application/input/adapters/lotmapper"
	"rutadeorigen/traceability/internal/application/input/dto/response"
	"rutadeorigen/traceability/internal/domain/lot"
)

func restoredEvent(details lot.StageDetails) (*lot.Event, lot.RestoreEventParams) {
	params := lot.RestoreEventParams{
		ID:           uuid.New(),
		LotID:        uuid.New(),
		Sequence:     2,
		EventType:    lot.EventTypeStage,
		Stage:        lot.StageFarm,
		EventDate:    time.Date(2026, time.October, 1, 0, 0, 0, 0, time.UTC),
		Note:         "Dried on raised beds",
		ActorID:      uuid.New(),
		ActorRole:    "producer",
		Details:      details,
		PreviousHash: "previous",
		Hash:         "current",
		RegisteredAt: time.Date(2026, time.October, 2, 15, 4, 5, 123456000, time.FixedZone("UTC-5", -5*3600)),
	}
	return lot.RestoreEvent(params), params
}

func TestToEventResponse(t *testing.T) {
	t.Run("maps every field", func(t *testing.T) {
		details := lot.FarmDetails{
			DryingMethod:      lot.DryingMethodAfricanBed,
			DryingDays:        15,
			MoisturePercent:   11.5,
			ParchmentWeightKg: 120.5,
		}
		e, p := restoredEvent(details)

		got := lotmapper.ToEventResponse(e)

		want := response.LotEvent{
			ID:           p.ID.String(),
			LotID:        p.LotID.String(),
			Sequence:     2,
			Type:         "stage",
			Stage:        "farm",
			EventDate:    "2026-10-01",
			Note:         "Dried on raised beds",
			ActorID:      p.ActorID.String(),
			ActorRole:    "producer",
			Details:      json.RawMessage(`{"drying_method":"african_bed","drying_days":15,"moisture_percent":11.5,"parchment_weight_kg":120.5}`),
			PreviousHash: "previous",
			Hash:         "current",
			RegisteredAt: "2026-10-02T20:04:05.123456Z",
		}
		if got.ID != want.ID || got.LotID != want.LotID || got.Sequence != want.Sequence {
			t.Errorf("unexpected identity: %+v", got)
		}
		if got.Type != want.Type || got.Stage != want.Stage || got.EventDate != want.EventDate {
			t.Errorf("unexpected classification: %+v", got)
		}
		if got.Note != want.Note || got.ActorID != want.ActorID || got.ActorRole != want.ActorRole {
			t.Errorf("unexpected note or actor: %+v", got)
		}
		if got.PreviousHash != want.PreviousHash || got.Hash != want.Hash {
			t.Errorf("unexpected hashes: %+v", got)
		}
		if got.RegisteredAt != want.RegisteredAt {
			t.Errorf("RegisteredAt: got %q, want %q", got.RegisteredAt, want.RegisteredAt)
		}
		if string(got.Details) != string(want.Details) {
			t.Errorf("Details: got %s, want %s", got.Details, want.Details)
		}
	})

	t.Run("serializes the details of every stage", func(t *testing.T) {
		tests := []struct {
			name    string
			details lot.StageDetails
			want    string
		}{
			{"roasting", lot.RoastingDetails{RoastLevel: lot.RoastLevelMedium, DurationMinutes: 14, GreenWeightKg: 100, RoastedWeightKg: 84},
				`{"roast_level":"medium","duration_minutes":14,"green_weight_kg":100,"roasted_weight_kg":84}`},
			{"packaging", lot.PackagingDetails{Grind: lot.GrindGround, PackageWeightG: 250, Units: 40, BestBefore: "2027-04-01"},
				`{"grind":"ground","package_weight_g":250,"units":40,"best_before":"2027-04-01"}`},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				e, _ := restoredEvent(tt.details)

				got := lotmapper.ToEventResponse(e)

				if string(got.Details) != tt.want {
					t.Errorf("got %s, want %s", got.Details, tt.want)
				}
			})
		}
	})

	t.Run("the response serializes to the expected JSON keys", func(t *testing.T) {
		e, _ := restoredEvent(lot.FarmDetails{DryingMethod: lot.DryingMethodSunPatio})

		raw, err := json.Marshal(lotmapper.ToEventResponse(e))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var decoded map[string]any
		if err := json.Unmarshal(raw, &decoded); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		for _, key := range []string{"id", "lot_id", "sequence", "type", "stage", "event_date", "note", "actor_id",
			"actor_role", "details", "previous_hash", "hash", "registered_at"} {
			if _, ok := decoded[key]; !ok {
				t.Errorf("missing key %q", key)
			}
		}
		if _, ok := decoded["details"].(map[string]any); !ok {
			t.Errorf("details must be an object, got %T", decoded["details"])
		}
	})

	t.Run("an event without a note omits it", func(t *testing.T) {
		e, p := restoredEvent(lot.FarmDetails{})
		p.Note = ""
		e = lot.RestoreEvent(p)

		raw, err := json.Marshal(lotmapper.ToEventResponse(e))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var decoded map[string]any
		if err := json.Unmarshal(raw, &decoded); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, ok := decoded["note"]; ok {
			t.Error("an empty note must be omitted")
		}
	})

	t.Run("an event without details serializes them as null", func(t *testing.T) {
		e, _ := restoredEvent(nil)

		got := lotmapper.ToEventResponse(e)

		if string(got.Details) != "null" {
			t.Errorf("got %s, want null", got.Details)
		}
	})

	t.Run("a default event maps to an empty response", func(t *testing.T) {
		got := lotmapper.ToEventResponse(lot.DefaultEvent())

		if got.ID != "" || got.Hash != "" || got.Details != nil || got.Sequence != 0 {
			t.Errorf("expected an empty response, got %+v", got)
		}
	})

	t.Run("a nil event maps to an empty response", func(t *testing.T) {
		got := lotmapper.ToEventResponse(nil)

		if got.ID != "" || got.Details != nil {
			t.Errorf("expected an empty response, got %+v", got)
		}
	})
}
