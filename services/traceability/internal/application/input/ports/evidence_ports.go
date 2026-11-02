package ports

import (
	"rutadeorigen/traceability/internal/application/input/dto/request"
	"rutadeorigen/traceability/internal/application/input/dto/response"
)

type RequestEvidenceUploadPort interface {
	InputPort[request.RequestEvidenceUpload, response.EvidenceUpload]
}

type CompleteEvidenceUploadPort interface {
	InputPort[request.CompleteEvidenceUpload, response.Evidence]
}

type ListEvidencesPort interface {
	InputPort[request.ListEvidences, response.EvidenceList]
}

type RecordEvidenceResultPort interface {
	InputPortWithoutReturn[request.RecordEvidenceResult]
}
