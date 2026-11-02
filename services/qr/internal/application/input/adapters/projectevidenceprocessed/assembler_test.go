package projectevidenceprocessed_test

import (
	"testing"

	"github.com/google/uuid"

	"rutadeorigen/qr/internal/application/input/adapters/projectevidenceprocessed"
	"rutadeorigen/qr/internal/application/input/dto/request"
	"rutadeorigen/qr/internal/platform/exception"
)

func validRequest() request.ProjectEvidenceProcessed {
	return request.ProjectEvidenceProcessed{
		MessageID:  uuid.NewString(),
		EventType:  " evidence.processed ",
		OccurredAt: "2026-11-02T10:00:00Z",
		Data: request.EvidenceProcessedData{
			EvidenceID:   uuid.NewString(),
			LotID:        uuid.NewString(),
			Stage:        " farm ",
			FinalKey:     " lot/ev/imagen.jpg ",
			ThumbnailKey: "lot/ev/miniatura.webp",
			ContentType:  "image/jpeg",
			ProcessedAt:  "2026-11-02T10:00:05Z",
		},
	}
}

func TestToDomainMapsAndTrims(t *testing.T) {
	req := validRequest()

	cmd, err := projectevidenceprocessed.Assembler{}.ToDomain(req)

	if err != nil {
		t.Fatal(err)
	}
	if cmd.EventType != "evidence.processed" || cmd.Stage != "farm" || cmd.FinalKey != "lot/ev/imagen.jpg" {
		t.Errorf("fields must be trimmed: %+v", cmd)
	}
	if cmd.EvidenceID.String() != req.Data.EvidenceID || cmd.ProcessedAt.IsZero() {
		t.Errorf("unexpected command %+v", cmd)
	}
}

func TestToDomainAllowsMissingProcessedAt(t *testing.T) {
	req := validRequest()
	req.Data.ProcessedAt = ""

	if _, err := (projectevidenceprocessed.Assembler{}).ToDomain(req); err != nil {
		t.Errorf("a missing processed time is filled by the domain, got %v", err)
	}
}

func TestToDomainRejectsInvalidInput(t *testing.T) {
	cases := map[string]func(r *request.ProjectEvidenceProcessed){
		"message id":   func(r *request.ProjectEvidenceProcessed) { r.MessageID = "nope" },
		"evidence id":  func(r *request.ProjectEvidenceProcessed) { r.Data.EvidenceID = "" },
		"lot id":       func(r *request.ProjectEvidenceProcessed) { r.Data.LotID = "x" },
		"processed at": func(r *request.ProjectEvidenceProcessed) { r.Data.ProcessedAt = "yesterday" },
	}

	for name, breakIt := range cases {
		t.Run(name, func(t *testing.T) {
			req := validRequest()
			breakIt(&req)

			_, err := projectevidenceprocessed.Assembler{}.ToDomain(req)

			if !exception.HasCode(err, exception.CodeEventInvalidPayload) {
				t.Errorf("want invalid payload, got %v", err)
			}
		})
	}
}
