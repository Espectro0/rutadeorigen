package projectlottransferred_test

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"rutadeorigen/qr/internal/application/input/adapters/projectlottransferred"
	"rutadeorigen/qr/internal/application/input/dto/request"
	"rutadeorigen/qr/internal/platform/exception"
)

func validRequest() request.ProjectLotTransferred {
	return request.ProjectLotTransferred{
		MessageID:  uuid.NewString(),
		EventType:  " lot.transferred ",
		Version:    4,
		OccurredAt: "2026-10-02T09:00:00Z",
		Data: request.LotTransferredData{
			LotID:       uuid.NewString(),
			CustodianID: uuid.NewString(),
		},
	}
}

func TestToDomainMapsTheRequest(t *testing.T) {
	req := validRequest()

	cmd, err := projectlottransferred.Assembler{}.ToDomain(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cmd.MessageID.String() != req.MessageID || cmd.LotID.String() != req.Data.LotID || cmd.CustodianID.String() != req.Data.CustodianID {
		t.Errorf("unexpected ids %+v", cmd)
	}
	if cmd.EventType != "lot.transferred" || cmd.Version != 4 || !cmd.OccurredAt.Equal(time.Date(2026, 10, 2, 9, 0, 0, 0, time.UTC)) {
		t.Errorf("unexpected values %+v", cmd)
	}
}

func TestToDomainAcceptsAMissingOccurredAt(t *testing.T) {
	req := validRequest()
	req.OccurredAt = ""

	if _, err := (projectlottransferred.Assembler{}).ToDomain(req); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestToDomainRejectsInvalidPayloads(t *testing.T) {
	for name, mutate := range map[string]func(*request.ProjectLotTransferred){
		"message id":   func(r *request.ProjectLotTransferred) { r.MessageID = "x" },
		"lot id":       func(r *request.ProjectLotTransferred) { r.Data.LotID = "" },
		"custodian id": func(r *request.ProjectLotTransferred) { r.Data.CustodianID = "x" },
		"occurred at":  func(r *request.ProjectLotTransferred) { r.OccurredAt = "yesterday" },
	} {
		t.Run(name, func(t *testing.T) {
			req := validRequest()
			mutate(&req)

			_, err := projectlottransferred.Assembler{}.ToDomain(req)

			if !exception.HasCode(err, exception.CodeEventInvalidPayload) {
				t.Errorf("got error %v, want %s", err, exception.CodeEventInvalidPayload)
			}
		})
	}
}
