package ports

import "rutadeorigen/processor/internal/application/input/dto/request"

type ProcessEvidencePort interface {
	InputPortWithoutReturn[request.ProcessEvidence]
}
