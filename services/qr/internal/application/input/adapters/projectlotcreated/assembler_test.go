package projectlotcreated_test

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"rutadeorigen/qr/internal/application/input/adapters/projectlotcreated"
	"rutadeorigen/qr/internal/application/input/dto/request"
	"rutadeorigen/qr/internal/application/input/dto/response"
	projectlotcreateduc "rutadeorigen/qr/internal/application/usecases/projectlotcreated"
	"rutadeorigen/qr/internal/domain/lotview"
	"rutadeorigen/qr/internal/platform/exception"
)

func validRequest() request.ProjectLotCreated {
	return request.ProjectLotCreated{
		MessageID:  uuid.NewString(),
		EventType:  "lot.created",
		Version:    1,
		OccurredAt: "2026-09-20T14:30:00Z",
		Data: request.LotCreatedData{
			LotID:       uuid.NewString(),
			Farm:        request.LotCreatedFarm{ID: uuid.NewString(), Name: "La Esperanza", Altitude: 1800},
			Producer:    request.LotCreatedActor{ID: uuid.NewString(), Name: "Maria Lopez"},
			CustodianID: uuid.NewString(),
			Code:        "LT-001",
			Variety:     "Castillo",
			Process:     "washed",
			HarvestDate: "2026-09-15",
			Stage:       "none",
			Status:      "active",
		},
	}
}

func TestAssemblerToDomain(t *testing.T) {
	t.Run("builds a command with the parsed values", func(t *testing.T) {
		req := validRequest()

		cmd, err := projectlotcreated.Assembler{}.ToDomain(req)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cmd.MessageID.String() != req.MessageID || cmd.EventType != "lot.created" {
			t.Errorf("unexpected message: %s %q", cmd.MessageID, cmd.EventType)
		}
		p := cmd.Lot
		if p.LotID.String() != req.Data.LotID || p.Code != "LT-001" || p.Variety != "Castillo" || p.Version != 1 {
			t.Errorf("unexpected lot params: %+v", p)
		}
		if p.Process != lotview.ProcessWashed || p.CurrentStage != lotview.StageNone || p.Status != lotview.StatusActive {
			t.Errorf("unexpected enums: %q %q %q", p.Process, p.CurrentStage, p.Status)
		}
		if p.FarmID.String() != req.Data.Farm.ID || p.FarmName != "La Esperanza" || p.FarmAltitude != 1800 {
			t.Errorf("unexpected farm: %+v", p)
		}
		if p.ProducerID.String() != req.Data.Producer.ID || p.ProducerName != "Maria Lopez" || p.CustodianID.String() != req.Data.CustodianID {
			t.Errorf("unexpected producer or custodian: %+v", p)
		}
		wantHarvest := time.Date(2026, time.September, 15, 0, 0, 0, 0, time.UTC)
		wantCreated := time.Date(2026, time.September, 20, 14, 30, 0, 0, time.UTC)
		if !p.HarvestDate.Equal(wantHarvest) || !p.CreatedAt.Equal(wantCreated) {
			t.Errorf("unexpected dates: %s %s", p.HarvestDate, p.CreatedAt)
		}
	})

	t.Run("accepts a missing occurred at", func(t *testing.T) {
		req := validRequest()
		req.OccurredAt = ""

		cmd, err := projectlotcreated.Assembler{}.ToDomain(req)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !cmd.Lot.CreatedAt.IsZero() {
			t.Errorf("CreatedAt: got %s, want the default date", cmd.Lot.CreatedAt)
		}
	})

	t.Run("enums are case insensitive", func(t *testing.T) {
		req := validRequest()
		req.Data.Process = "  NATURAL "
		req.Data.Stage = "Packaging"
		req.Data.Status = "VOIDED"

		cmd, err := projectlotcreated.Assembler{}.ToDomain(req)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cmd.Lot.Process != lotview.ProcessNatural || cmd.Lot.CurrentStage != lotview.StagePackaging || cmd.Lot.Status != lotview.StatusVoided {
			t.Errorf("unexpected enums: %q %q %q", cmd.Lot.Process, cmd.Lot.CurrentStage, cmd.Lot.Status)
		}
	})

	invalid := []struct {
		name   string
		mutate func(r *request.ProjectLotCreated)
	}{
		{"empty message id", func(r *request.ProjectLotCreated) { r.MessageID = "" }},
		{"malformed message id", func(r *request.ProjectLotCreated) { r.MessageID = "not-a-uuid" }},
		{"nil message id", func(r *request.ProjectLotCreated) { r.MessageID = uuid.Nil.String() }},
		{"malformed occurred at", func(r *request.ProjectLotCreated) { r.OccurredAt = "yesterday" }},
	}

	for _, tc := range invalid {
		t.Run(tc.name, func(t *testing.T) {
			req := validRequest()
			tc.mutate(&req)

			cmd, err := projectlotcreated.Assembler{}.ToDomain(req)

			if !exception.HasCode(err, exception.CodeEventInvalidPayload) {
				t.Fatalf("got error %v, want code %q", err, exception.CodeEventInvalidPayload)
			}
			if cmd.MessageID != uuid.Nil {
				t.Errorf("on error the command must be empty, got %+v", cmd)
			}
		})
	}

	t.Run("leaves the lot fields for the domain to validate", func(t *testing.T) {
		req := validRequest()
		req.Data.LotID = "not-a-uuid"
		req.Data.Process = "fermented"
		req.Data.HarvestDate = "15/09/2026"

		cmd, err := projectlotcreated.Assembler{}.ToDomain(req)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cmd.Lot.LotID != uuid.Nil || cmd.Lot.Process != lotview.ProcessUnknown || !cmd.Lot.HarvestDate.IsZero() {
			t.Errorf("malformed values must become defaults, got %+v", cmd.Lot)
		}
	})
}

func TestAssemblerToDTO(t *testing.T) {
	if got := (projectlotcreated.Assembler{}).ToDTO(projectlotcreateduc.Command{}); got != (response.Empty{}) {
		t.Errorf("got %+v, want empty response", got)
	}
}
