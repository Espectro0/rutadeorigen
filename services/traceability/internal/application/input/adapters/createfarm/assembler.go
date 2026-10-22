package createfarm

import (
	"rutadeorigen/traceability/internal/application/input/adapters/farmmapper"
	"rutadeorigen/traceability/internal/application/input/assembler"
	"rutadeorigen/traceability/internal/application/input/dto/request"
	"rutadeorigen/traceability/internal/application/input/dto/response"
	"rutadeorigen/traceability/internal/domain/farm"
)

var _ assembler.DTOAssembler[*farm.Farm, request.CreateFarm, response.Farm] = Assembler{}

type Assembler struct{}

func (Assembler) ToDomain(req request.CreateFarm) (*farm.Farm, error) {
	return farm.Draft(req.Name, req.Altitude), nil
}

func (Assembler) ToDTO(f *farm.Farm) response.Farm {
	return farmmapper.ToResponse(f)
}
