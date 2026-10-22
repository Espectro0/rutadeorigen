package ports

import (
	"rutadeorigen/qr/internal/application/input/dto/request"
	"rutadeorigen/qr/internal/application/input/dto/response"
)

type ResolveScanPort interface {
	InputPort[request.ResolveScan, response.ScanRedirect]
}
