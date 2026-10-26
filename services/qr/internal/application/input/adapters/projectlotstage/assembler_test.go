package projectlotstage_test

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"rutadeorigen/qr/internal/application/input/adapters/projectlotstage"
	"rutadeorigen/qr/internal/application/input/dto/request"
	"rutadeorigen/qr/internal/application/input/dto/response"
	"rutadeorigen/qr/internal/domain/lotview"
	"rutadeorigen/qr/internal/platform/exception"
)

func validRequest() request.ProjectLotStage {
	return request.ProjectLotStage{
		MessageID:  uuid.NewString(),
		EventType:  "lot.stage_registered",
		Version:    3,
		OccurredAt: "2026-09-20T14:30:00Z",
		Data: request.LotStageData{
			LotID:     uuid.NewString(),
			Stage:     "roasting",
			Status:    "active",
			EventID:   uuid.NewString(),
			Sequence:  2,
			EventDate: "2026-09-19",
		},
	}
}

func TestAssemblerToDomain(t *testing.T) {
	t.Run("builds a command with the parsed values", func(t *testing.T) {
		req := validRequest()

		cmd, err := projectlotstage.Assembler{}.ToDomain(req)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cmd.MessageID.String() != req.MessageID || cmd.EventType != "lot.stage_registered" {
			t.Errorf("unexpected message: %s %q", cmd.MessageID, cmd.EventType)
		}
		if cmd.LotID.String() != req.Data.LotID || cmd.Stage != lotview.StageRoasting || cmd.Version != 3 {
			t.Errorf("unexpected lot data: %s %q %d", cmd.LotID, cmd.Stage, cmd.Version)
		}
		wantOccurred := time.Date(2026, time.September, 20, 14, 30, 0, 0, time.UTC)
		if !cmd.OccurredAt.Equal(wantOccurred) {
			t.Errorf("OccurredAt: got %s, want %s", cmd.OccurredAt, wantOccurred)
		}
	})

	t.Run("normalizes the event type, the identifiers and the stage", func(t *testing.T) {
		req := validRequest()
		messageID, lotID := uuid.New(), uuid.New()
		req.MessageID = "  " + messageID.String() + " "
		req.EventType = "  lot.stage_registered  "
		req.Data.LotID = " " + lotID.String() + "  "
		req.Data.Stage = "  Packaging "

		cmd, err := projectlotstage.Assembler{}.ToDomain(req)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cmd.MessageID != messageID || cmd.LotID != lotID {
			t.Errorf("unexpected identifiers: %s %s", cmd.MessageID, cmd.LotID)
		}
		if cmd.EventType != "lot.stage_registered" || cmd.Stage != lotview.StagePackaging {
			t.Errorf("unexpected values: %q %q", cmd.EventType, cmd.Stage)
		}
	})

	t.Run("accepts a missing occurred at", func(t *testing.T) {
		req := validRequest()
		req.OccurredAt = ""

		cmd, err := projectlotstage.Assembler{}.ToDomain(req)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !cmd.OccurredAt.IsZero() {
			t.Errorf("OccurredAt: got %s, want the default date", cmd.OccurredAt)
		}
	})

	t.Run("converts the occurred at to UTC", func(t *testing.T) {
		req := validRequest()
		req.OccurredAt = "2026-09-20T09:30:00-05:00"

		cmd, err := projectlotstage.Assembler{}.ToDomain(req)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := time.Date(2026, time.September, 20, 14, 30, 0, 0, time.UTC)
		if cmd.OccurredAt.Location() != time.UTC || !cmd.OccurredAt.Equal(want) {
			t.Errorf("OccurredAt: got %s, want %s in UTC", cmd.OccurredAt, want)
		}
	})

	t.Run("leaves the stage validation to the domain", func(t *testing.T) {
		for _, stage := range []string{"", "drying", "none"} {
			req := validRequest()
			req.Data.Stage = stage

			cmd, err := projectlotstage.Assembler{}.ToDomain(req)

			if err != nil {
				t.Fatalf("stage %q: unexpected error: %v", stage, err)
			}
			if stage == "none" && cmd.Stage != lotview.StageNone {
				t.Errorf("stage %q: got %q", stage, cmd.Stage)
			}
			if stage != "none" && cmd.Stage != lotview.StageUnknown {
				t.Errorf("stage %q: got %q, want %q", stage, cmd.Stage, lotview.StageUnknown)
			}
		}
	})
}

func TestAssemblerToDomainRejectsInvalidPayloads(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(r *request.ProjectLotStage)
	}{
		{"empty message ID", func(r *request.ProjectLotStage) { r.MessageID = "" }},
		{"malformed message ID", func(r *request.ProjectLotStage) { r.MessageID = "not-a-uuid" }},
		{"empty lot ID", func(r *request.ProjectLotStage) { r.Data.LotID = "" }},
		{"malformed lot ID", func(r *request.ProjectLotStage) { r.Data.LotID = "not-a-uuid" }},
		{"zero lot ID", func(r *request.ProjectLotStage) { r.Data.LotID = uuid.Nil.String() }},
		{"malformed occurred at", func(r *request.ProjectLotStage) { r.OccurredAt = "yesterday" }},
		{"occurred at without time", func(r *request.ProjectLotStage) { r.OccurredAt = "2026-09-20" }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := validRequest()
			tt.mutate(&req)

			_, err := projectlotstage.Assembler{}.ToDomain(req)

			if !exception.HasCode(err, exception.CodeEventInvalidPayload) {
				t.Errorf("got %v, want %s", err, exception.CodeEventInvalidPayload)
			}
		})
	}
}

func TestAssemblerToDTO(t *testing.T) {
	cmd, err := projectlotstage.Assembler{}.ToDomain(validRequest())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := (projectlotstage.Assembler{}).ToDTO(cmd); got != (response.Empty{}) {
		t.Errorf("got %+v, want an empty response", got)
	}
}
