package createlot_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"rutadeorigen/traceability/internal/application/input/adapters/createlot"
	"rutadeorigen/traceability/internal/application/input/dto/request"
	"rutadeorigen/traceability/internal/application/input/dto/response"
	"rutadeorigen/traceability/internal/domain/lot"
	"rutadeorigen/traceability/internal/platform/exception"
)

func validRequest(farmID uuid.UUID) request.CreateLot {
	return request.CreateLot{
		FarmID:      farmID.String(),
		Code:        "LOT-001",
		Variety:     "Castillo",
		Process:     "washed",
		HarvestDate: "2026-09-15",
	}
}

func TestAssemblerToDomain(t *testing.T) {
	t.Run("builds a draft with the parsed values", func(t *testing.T) {
		farmID := uuid.New()

		draft, err := createlot.Assembler{}.ToDomain(validRequest(farmID))

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if draft.FarmID() != farmID {
			t.Errorf("FarmID: got %s, want %s", draft.FarmID(), farmID)
		}
		if draft.Code() != "LOT-001" {
			t.Errorf("Code: got %q, want %q", draft.Code(), "LOT-001")
		}
		if draft.Variety() != "Castillo" {
			t.Errorf("Variety: got %q, want %q", draft.Variety(), "Castillo")
		}
		if draft.Process() != lot.ProcessWashed {
			t.Errorf("Process: got %q, want %q", draft.Process(), lot.ProcessWashed)
		}
		wantDate := time.Date(2026, time.September, 15, 0, 0, 0, 0, time.UTC)
		if !draft.HarvestDate().Equal(wantDate) {
			t.Errorf("HarvestDate: got %s, want %s", draft.HarvestDate(), wantDate)
		}
	})

	t.Run("draft is not a persisted lot", func(t *testing.T) {
		draft, _ := createlot.Assembler{}.ToDomain(validRequest(uuid.New()))

		if !draft.IsDefault() {
			t.Error("a draft must not have an ID yet")
		}
	})

	t.Run("process is case insensitive", func(t *testing.T) {
		req := validRequest(uuid.New())
		req.Process = "  NATURAL "

		draft, err := createlot.Assembler{}.ToDomain(req)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if draft.Process() != lot.ProcessNatural {
			t.Errorf("Process: got %q, want %q", draft.Process(), lot.ProcessNatural)
		}
	})

	invalid := []struct {
		name    string
		mutate  func(r *request.CreateLot)
		wantErr error
	}{
		{"empty farm id", func(r *request.CreateLot) { r.FarmID = "" }, createlot.ErrInvalidFarmID},
		{"malformed farm id", func(r *request.CreateLot) { r.FarmID = "not-a-uuid" }, createlot.ErrInvalidFarmID},
		{"nil farm id", func(r *request.CreateLot) { r.FarmID = uuid.Nil.String() }, createlot.ErrInvalidFarmID},
		{"empty harvest date", func(r *request.CreateLot) { r.HarvestDate = "" }, createlot.ErrInvalidHarvestDate},
		{"wrong date format", func(r *request.CreateLot) { r.HarvestDate = "15/09/2026" }, createlot.ErrInvalidHarvestDate},
		{"impossible date", func(r *request.CreateLot) { r.HarvestDate = "2026-02-30" }, createlot.ErrInvalidHarvestDate},
		{"empty process", func(r *request.CreateLot) { r.Process = "" }, createlot.ErrInvalidProcess},
		{"unknown process", func(r *request.CreateLot) { r.Process = "fermented" }, createlot.ErrInvalidProcess},
	}

	for _, tc := range invalid {
		t.Run(tc.name, func(t *testing.T) {
			req := validRequest(uuid.New())
			tc.mutate(&req)

			draft, err := createlot.Assembler{}.ToDomain(req)

			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("got error %v, want %v", err, tc.wantErr)
			}
			if !draft.IsDefault() || draft.FarmID() != uuid.Nil {
				t.Error("on error the assembler must return lot.Default()")
			}
			if ex, ok := exception.From(err); !ok || ex.Kind() != exception.KindValidation {
				t.Errorf("error must be a validation exception, got %v", err)
			}
		})
	}
}

func TestAssemblerToDTO(t *testing.T) {
	t.Run("maps the lot to a response", func(t *testing.T) {
		id := uuid.New()
		l := lot.Restore(lot.RestoreParams{
			ID:           id,
			FarmID:       uuid.New(),
			CustodianID:  uuid.New(),
			Code:         "LOT-001",
			Variety:      "Castillo",
			Process:      lot.ProcessHoney,
			HarvestDate:  time.Date(2026, time.September, 15, 0, 0, 0, 0, time.UTC),
			CurrentStage: lot.StageNone,
			Status:       lot.StatusActive,
			Version:      1,
			CreatedAt:    time.Date(2026, time.September, 20, 0, 0, 0, 0, time.UTC),
		})

		got := createlot.Assembler{}.ToDTO(l)

		if got.ID != id.String() || got.Process != "honey" || got.Stage != "none" {
			t.Errorf("unexpected response: %+v", got)
		}
	})

	t.Run("default lot returns an empty response", func(t *testing.T) {
		if got := (createlot.Assembler{}).ToDTO(lot.Default()); got != (response.Lot{}) {
			t.Errorf("got %+v, want empty response", got)
		}
	})
}
