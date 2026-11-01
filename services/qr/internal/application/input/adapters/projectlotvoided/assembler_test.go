package projectlotvoided_test

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"rutadeorigen/qr/internal/application/input/adapters/projectlotvoided"
	"rutadeorigen/qr/internal/application/input/dto/request"
	"rutadeorigen/qr/internal/platform/exception"
)

func validRequest() request.ProjectLotVoided {
	return request.ProjectLotVoided{
		MessageID:  uuid.NewString(),
		EventType:  " lot.voided ",
		Version:    5,
		OccurredAt: "2026-10-02T09:00:00Z",
		Data: request.LotVoidedData{
			LotID:  uuid.NewString(),
			Reason: "  Lote contaminado ",
		},
	}
}

func TestToDomainMapsTheRequest(t *testing.T) {
	req := validRequest()

	cmd, err := projectlotvoided.Assembler{}.ToDomain(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cmd.MessageID.String() != req.MessageID || cmd.LotID.String() != req.Data.LotID {
		t.Errorf("unexpected ids %+v", cmd)
	}
	if cmd.EventType != "lot.voided" || cmd.Reason != "Lote contaminado" || cmd.Version != 5 ||
		!cmd.OccurredAt.Equal(time.Date(2026, 10, 2, 9, 0, 0, 0, time.UTC)) {
		t.Errorf("unexpected values %+v", cmd)
	}
}

func TestToDomainRejectsInvalidPayloads(t *testing.T) {
	for name, mutate := range map[string]func(*request.ProjectLotVoided){
		"message id":  func(r *request.ProjectLotVoided) { r.MessageID = "" },
		"lot id":      func(r *request.ProjectLotVoided) { r.Data.LotID = "x" },
		"occurred at": func(r *request.ProjectLotVoided) { r.OccurredAt = "yesterday" },
	} {
		t.Run(name, func(t *testing.T) {
			req := validRequest()
			mutate(&req)

			_, err := projectlotvoided.Assembler{}.ToDomain(req)

			if !exception.HasCode(err, exception.CodeEventInvalidPayload) {
				t.Errorf("got error %v, want %s", err, exception.CodeEventInvalidPayload)
			}
		})
	}
}
