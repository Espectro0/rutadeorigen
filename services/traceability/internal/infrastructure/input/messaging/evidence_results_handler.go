package messaging

import (
	"context"
	"encoding/json"

	libmessaging "rutadeorigen/lib/messaging"
	"rutadeorigen/traceability/internal/application/input/dto/request"
	"rutadeorigen/traceability/internal/application/input/ports"
	"rutadeorigen/traceability/internal/platform/exception"
)

const (
	EventEvidenceProcessed = "evidence.processed"
	EventEvidenceRejected  = "evidence.rejected"
)

type EvidenceResultsHandler struct {
	record ports.RecordEvidenceResultPort
}

func NewEvidenceResultsHandler(record ports.RecordEvidenceResultPort) *EvidenceResultsHandler {
	return &EvidenceResultsHandler{record: record}
}

type resultEnvelope struct {
	Data struct {
		EvidenceID   string `json:"evidence_id"`
		SHA256       string `json:"sha256"`
		FinalKey     string `json:"final_key"`
		ThumbnailKey string `json:"thumbnail_key"`
		Reason       string `json:"reason"`
		ProcessedAt  string `json:"processed_at"`
	} `json:"data"`
}

func (h *EvidenceResultsHandler) Handle(ctx context.Context, d libmessaging.Delivery) error {
	if d.Type != EventEvidenceProcessed && d.Type != EventEvidenceRejected {
		return nil
	}

	var env resultEnvelope
	if err := json.Unmarshal(d.Body, &env); err != nil {
		return libmessaging.Permanent(exception.New(exception.CodeEventInvalidPayload).WithCause(err))
	}

	err := h.record.Execute(ctx, request.RecordEvidenceResult{
		Type:         d.Type,
		EvidenceID:   env.Data.EvidenceID,
		SHA256:       env.Data.SHA256,
		FinalKey:     env.Data.FinalKey,
		ThumbnailKey: env.Data.ThumbnailKey,
		Reason:       env.Data.Reason,
		ProcessedAt:  env.Data.ProcessedAt,
	})
	if isPermanent(err) {
		return libmessaging.Permanent(err)
	}
	return err
}

// An event that can never succeed must not be retried.
func isPermanent(err error) bool {
	return exception.HasCode(err, exception.CodeEvidenceNotFound) ||
		exception.HasCode(err, exception.CodeEvidenceInvalidResult) ||
		exception.HasCode(err, exception.CodeEventInvalidPayload)
}
