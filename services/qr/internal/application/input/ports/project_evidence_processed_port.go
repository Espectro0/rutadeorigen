package ports

import (
	"rutadeorigen/qr/internal/application/input/dto/request"
	"rutadeorigen/qr/internal/application/input/dto/response"
)

type ProjectEvidenceProcessedPort interface {
	InputPort[request.ProjectEvidenceProcessed, response.Empty]
}
