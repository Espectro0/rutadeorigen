package evidencemapper

import (
	"time"

	"rutadeorigen/traceability/internal/application/input/dto/response"
	"rutadeorigen/traceability/internal/domain/evidence"
	"rutadeorigen/traceability/internal/platform/helper"
)

func ToResponse(e *evidence.Evidence) response.Evidence {
	if e.IsDefault() {
		return response.Evidence{}
	}

	processedAt := ""
	if !helper.IsDefaultDate(e.ProcessedAt()) {
		processedAt = e.ProcessedAt().Format(time.RFC3339)
	}

	return response.Evidence{
		ID:              e.ID().String(),
		LotID:           e.LotID().String(),
		Stage:           e.Stage().String(),
		UploadedBy:      e.UploadedBy().String(),
		FileName:        e.FileName(),
		ContentType:     e.ContentType().String(),
		SizeBytes:       e.SizeBytes(),
		Status:          e.Status().String(),
		SHA256:          e.SHA256(),
		RejectionReason: e.RejectionReason(),
		CreatedAt:       e.CreatedAt().Format(time.RFC3339),
		ProcessedAt:     processedAt,
	}
}
