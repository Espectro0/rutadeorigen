package registerstage_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"

	"rutadeorigen/traceability/internal/application/input/adapters/registerstage"
	"rutadeorigen/traceability/internal/application/input/dto/request"
	registerstageuc "rutadeorigen/traceability/internal/application/usecases/registerstage"
	"rutadeorigen/traceability/internal/domain/lot"
	"rutadeorigen/traceability/internal/platform/exception"
)

const (
	farmDetailsJSON      = `{"drying_method":"sun_patio","drying_days":15,"moisture_percent":11.5,"parchment_weight_kg":120.5}`
	roastingDetailsJSON  = `{"roast_level":"medium","duration_minutes":14,"green_weight_kg":100,"roasted_weight_kg":84}`
	packagingDetailsJSON = `{"grind":"whole_bean","package_weight_g":250,"units":40,"best_before":"2027-04-01"}`
)

func validRequest() request.RegisterStage {
	return request.RegisterStage{
		LotID:     uuid.NewString(),
		IfMatch:   `"3"`,
		Stage:     "farm",
		EventDate: "2026-10-01",
		Note:      "Dried on raised beds",
		Details:   json.RawMessage(farmDetailsJSON),
	}
}

func TestAssemblerToCommand(t *testing.T) {
	t.Run("maps a farm request", func(t *testing.T) {
		req := validRequest()

		got, err := registerstage.Assembler{}.ToCommand(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if got.LotID.String() != req.LotID || got.ExpectedVersion != 3 || got.Stage != lot.StageFarm {
			t.Errorf("unexpected command: %+v", got)
		}
		if !got.EventDate.Equal(time.Date(2026, time.October, 1, 0, 0, 0, 0, time.UTC)) {
			t.Errorf("EventDate: got %v", got.EventDate)
		}
		if got.Note != "Dried on raised beds" {
			t.Errorf("Note: got %q", got.Note)
		}
		want := lot.FarmDetails{
			DryingMethod:      lot.DryingMethodSunPatio,
			DryingDays:        15,
			MoisturePercent:   11.5,
			ParchmentWeightKg: 120.5,
		}
		if got.Details != want {
			t.Errorf("Details: got %+v, want %+v", got.Details, want)
		}
	})

	t.Run("maps a roasting request", func(t *testing.T) {
		req := validRequest()
		req.Stage = "roasting"
		req.Details = json.RawMessage(roastingDetailsJSON)

		got, err := registerstage.Assembler{}.ToCommand(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		want := lot.RoastingDetails{RoastLevel: lot.RoastLevelMedium, DurationMinutes: 14, GreenWeightKg: 100, RoastedWeightKg: 84}
		if got.Stage != lot.StageRoasting || got.Details != want {
			t.Errorf("unexpected command: %+v", got)
		}
	})

	t.Run("maps a packaging request", func(t *testing.T) {
		req := validRequest()
		req.Stage = "packaging"
		req.Details = json.RawMessage(packagingDetailsJSON)

		got, err := registerstage.Assembler{}.ToCommand(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		want := lot.PackagingDetails{Grind: lot.GrindWholeBean, PackageWeightG: 250, Units: 40, BestBefore: "2027-04-01"}
		if got.Stage != lot.StagePackaging || got.Details != want {
			t.Errorf("unexpected command: %+v", got)
		}
	})

	t.Run("the details type matches the stage", func(t *testing.T) {
		for stage, details := range map[string]string{
			"farm": farmDetailsJSON, "roasting": roastingDetailsJSON, "packaging": packagingDetailsJSON,
		} {
			req := validRequest()
			req.Stage = stage
			req.Details = json.RawMessage(details)

			got, err := registerstage.Assembler{}.ToCommand(req)
			if err != nil {
				t.Fatalf("%s: unexpected error: %v", stage, err)
			}
			if got.Details.Stage().String() != stage {
				t.Errorf("%s: details belong to stage %q", stage, got.Details.Stage())
			}
		}
	})

	t.Run("normalizes the stage and the enums", func(t *testing.T) {
		req := validRequest()
		req.Stage = "  FARM  "
		req.Details = json.RawMessage(`{"drying_method":" African_Bed ","drying_days":15,"moisture_percent":11.5,"parchment_weight_kg":1}`)

		got, err := registerstage.Assembler{}.ToCommand(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if got.Stage != lot.StageFarm {
			t.Errorf("Stage: got %q", got.Stage)
		}
		if got.Details.(lot.FarmDetails).DryingMethod != lot.DryingMethodAfricanBed {
			t.Errorf("DryingMethod: got %q", got.Details.(lot.FarmDetails).DryingMethod)
		}
	})

	t.Run("an unknown enum becomes unknown so the domain rejects it", func(t *testing.T) {
		req := validRequest()
		req.Details = json.RawMessage(`{"drying_method":"oven","drying_days":15,"moisture_percent":11.5,"parchment_weight_kg":1}`)

		got, err := registerstage.Assembler{}.ToCommand(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if got.Details.(lot.FarmDetails).DryingMethod != lot.DryingMethodUnknown {
			t.Errorf("DryingMethod: got %q", got.Details.(lot.FarmDetails).DryingMethod)
		}
		if err := got.Details.Validate(got.EventDate); !exception.HasCode(err, exception.CodeStageInvalidDryingMethod) {
			t.Errorf("got %v, want code %q", err, exception.CodeStageInvalidDryingMethod)
		}
	})

	t.Run("missing details fields are left to the domain", func(t *testing.T) {
		req := validRequest()
		req.Details = json.RawMessage(`{}`)

		got, err := registerstage.Assembler{}.ToCommand(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if err := got.Details.Validate(got.EventDate); !exception.HasCode(err, exception.CodeStageInvalidDryingMethod) {
			t.Errorf("got %v, want code %q", err, exception.CodeStageInvalidDryingMethod)
		}
	})

	t.Run("the note is passed as received", func(t *testing.T) {
		req := validRequest()
		req.Note = "  spaced  "

		got, err := registerstage.Assembler{}.ToCommand(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if got.Note != "  spaced  " {
			t.Errorf("Note: got %q", got.Note)
		}
	})
}

func TestAssemblerToCommandIfMatch(t *testing.T) {
	valid := map[string]int{
		`3`:       3,
		`"3"`:     3,
		`W/"3"`:   3,
		` "12" `:  12,
		`"1"`:     1,
		`"00007"`: 7,
	}
	for header, want := range valid {
		t.Run("accepts "+header, func(t *testing.T) {
			req := validRequest()
			req.IfMatch = header

			got, err := registerstage.Assembler{}.ToCommand(req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.ExpectedVersion != want {
				t.Errorf("got %d, want %d", got.ExpectedVersion, want)
			}
		})
	}

	for _, header := range []string{"", "   ", "abc", "0", `"0"`, `"-1"`, `""`, `"1.5"`, `*`, `"1","2"`} {
		t.Run("rejects "+header, func(t *testing.T) {
			req := validRequest()
			req.IfMatch = header

			_, err := registerstage.Assembler{}.ToCommand(req)
			if !exception.HasCode(err, exception.CodeLotInvalidIfMatch) {
				t.Errorf("got %v, want code %q", err, exception.CodeLotInvalidIfMatch)
			}
		})
	}
}

func TestAssemblerToCommandErrors(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*request.RegisterStage)
		want   exception.Code
	}{
		{"empty lot ID", func(r *request.RegisterStage) { r.LotID = "" }, exception.CodeLotNotFound},
		{"invalid lot ID", func(r *request.RegisterStage) { r.LotID = "not-a-uuid" }, exception.CodeLotNotFound},
		{"nil lot ID", func(r *request.RegisterStage) { r.LotID = uuid.Nil.String() }, exception.CodeLotNotFound},
		{"missing If-Match", func(r *request.RegisterStage) { r.IfMatch = "" }, exception.CodeLotInvalidIfMatch},
		{"empty stage", func(r *request.RegisterStage) { r.Stage = "" }, exception.CodeLotInvalidStage},
		{"unknown stage", func(r *request.RegisterStage) { r.Stage = "export" }, exception.CodeLotInvalidStage},
		{"stage none", func(r *request.RegisterStage) { r.Stage = "none" }, exception.CodeLotInvalidStage},
		{"empty event date", func(r *request.RegisterStage) { r.EventDate = "" }, exception.CodeLotInvalidEventDate},
		{"event date with wrong format", func(r *request.RegisterStage) { r.EventDate = "01/10/2026" }, exception.CodeLotInvalidEventDate},
		{"event date with time", func(r *request.RegisterStage) { r.EventDate = "2026-10-01T10:00:00Z" }, exception.CodeLotInvalidEventDate},
		{"impossible event date", func(r *request.RegisterStage) { r.EventDate = "2026-02-31" }, exception.CodeLotInvalidEventDate},
		{"missing details", func(r *request.RegisterStage) { r.Details = nil }, exception.CodeStageInvalidDetails},
		{"empty details", func(r *request.RegisterStage) { r.Details = json.RawMessage(`  `) }, exception.CodeStageInvalidDetails},
		{"null details", func(r *request.RegisterStage) { r.Details = json.RawMessage(`null`) }, exception.CodeStageInvalidDetails},
		{"details as an array", func(r *request.RegisterStage) { r.Details = json.RawMessage(`[]`) }, exception.CodeStageInvalidDetails},
		{"details as a string", func(r *request.RegisterStage) { r.Details = json.RawMessage(`"farm"`) }, exception.CodeStageInvalidDetails},
		{"details with invalid JSON", func(r *request.RegisterStage) { r.Details = json.RawMessage(`{"drying_days":`) }, exception.CodeStageInvalidDetails},
		{"details with an unknown field", func(r *request.RegisterStage) {
			r.Details = json.RawMessage(`{"drying_method":"sun_patio","altitude":1500}`)
		}, exception.CodeStageInvalidDetails},
		{"details of another stage", func(r *request.RegisterStage) { r.Details = json.RawMessage(roastingDetailsJSON) }, exception.CodeStageInvalidDetails},
		{"details with a wrong type", func(r *request.RegisterStage) {
			r.Details = json.RawMessage(`{"drying_method":"sun_patio","drying_days":"ten"}`)
		}, exception.CodeStageInvalidDetails},
		{"details with a fractional integer", func(r *request.RegisterStage) {
			r.Details = json.RawMessage(`{"drying_method":"sun_patio","drying_days":1.5}`)
		}, exception.CodeStageInvalidDetails},
		{"details followed by more JSON", func(r *request.RegisterStage) { r.Details = json.RawMessage(farmDetailsJSON + ` {}`) }, exception.CodeStageInvalidDetails},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := validRequest()
			tt.mutate(&req)

			got, err := registerstage.Assembler{}.ToCommand(req)
			if !exception.HasCode(err, tt.want) {
				t.Fatalf("got %v, want code %q", err, tt.want)
			}
			if got != (registerstageuc.Command{}) {
				t.Errorf("expected the zero command on error, got %+v", got)
			}
		})
	}
}

func TestAssemblerToCommandReportsTheFirstError(t *testing.T) {
	req := request.RegisterStage{}

	_, err := registerstage.Assembler{}.ToCommand(req)

	if !exception.HasCode(err, exception.CodeLotNotFound) {
		t.Errorf("got %v, want code %q", err, exception.CodeLotNotFound)
	}
}

func TestAssemblerToDTO(t *testing.T) {
	event, err := lot.NewStageEvent(lot.StageEventParams{
		LotID:       uuid.New(),
		HarvestDate: time.Now().AddDate(0, -1, 0),
		Stage:       lot.StageFarm,
		EventDate:   time.Now().AddDate(0, 0, -1),
		Note:        "Dried on raised beds",
		ActorID:     uuid.New(),
		ActorRole:   "producer",
		Details: lot.FarmDetails{
			DryingMethod:      lot.DryingMethodSunPatio,
			DryingDays:        15,
			MoisturePercent:   11.5,
			ParchmentWeightKg: 120.5,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Run("maps the event and the new lot version", func(t *testing.T) {
		got := registerstage.Assembler{}.ToDTO(registerstageuc.Result{Event: event, LotVersion: 2})

		if got.LotVersion != 2 {
			t.Errorf("LotVersion: got %d, want 2", got.LotVersion)
		}
		if got.Event.ID != event.ID().String() || got.Event.Hash != event.Hash() || got.Event.Sequence != 1 {
			t.Errorf("unexpected event: %+v", got.Event)
		}
		if got.Event.Stage != "farm" || got.Event.Type != "stage" {
			t.Errorf("unexpected classification: %+v", got.Event)
		}
	})

	t.Run("a default event maps to an empty event", func(t *testing.T) {
		got := registerstage.Assembler{}.ToDTO(registerstageuc.Result{Event: lot.DefaultEvent()})

		if got.Event.ID != "" || got.LotVersion != 0 {
			t.Errorf("expected an empty response, got %+v", got)
		}
	})
}
