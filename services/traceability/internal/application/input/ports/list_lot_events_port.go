package ports

import (
	"rutadeorigen/traceability/internal/application/input/dto/request"
	"rutadeorigen/traceability/internal/application/input/dto/response"
)

type ListLotEventsPort interface {
	InputPort[request.ListLotEvents, response.LotEventList]
}
