package ports

import "rutadeorigen/traceability/internal/application/input/dto/request"

type DeleteFarmPort interface {
	InputPortWithoutReturn[request.DeleteFarm]
}
