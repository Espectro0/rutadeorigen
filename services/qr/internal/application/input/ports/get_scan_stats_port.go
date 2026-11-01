package ports

import (
	"rutadeorigen/qr/internal/application/input/dto/request"
	"rutadeorigen/qr/internal/application/input/dto/response"
)

type GetScanStatsPort interface {
	InputPort[request.GetScanStats, response.ScanStats]
}
