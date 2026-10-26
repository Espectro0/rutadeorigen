package ports

import (
	"rutadeorigen/qr/internal/application/input/dto/request"
	"rutadeorigen/qr/internal/application/input/dto/response"
)

type ProjectLotStagePort interface {
	InputPort[request.ProjectLotStage, response.Empty]
}
