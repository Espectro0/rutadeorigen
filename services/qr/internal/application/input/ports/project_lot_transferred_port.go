package ports

import (
	"rutadeorigen/qr/internal/application/input/dto/request"
	"rutadeorigen/qr/internal/application/input/dto/response"
)

type ProjectLotTransferredPort interface {
	InputPort[request.ProjectLotTransferred, response.Empty]
}
