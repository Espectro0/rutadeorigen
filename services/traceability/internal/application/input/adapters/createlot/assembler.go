package createlot

import (
	"rutadeorigen/traceability/internal/application/input/adapters/lotmapper"
	"rutadeorigen/traceability/internal/application/input/assembler"
	"rutadeorigen/traceability/internal/application/input/dto/request"
	"rutadeorigen/traceability/internal/application/input/dto/response"
	"rutadeorigen/traceability/internal/domain/lot"
	"rutadeorigen/traceability/internal/platform/exception"
	"rutadeorigen/traceability/internal/platform/helper"
)

var _ assembler.DTOAssembler[*lot.Lot, request.CreateLot, response.Lot] = Assembler{}

type Assembler struct{}

func (Assembler) ToDomain(req request.CreateLot) (*lot.Lot, error) {
	farmID := helper.ParseUUID(req.FarmID)
	if helper.IsDefaultUUID(farmID) {
		return lot.Default(), exception.New(exception.CodeLotInvalidFarmID)
	}

	harvestDate := helper.ParseDate(req.HarvestDate)
	if helper.IsDefaultDate(harvestDate) {
		return lot.Default(), exception.New(exception.CodeLotInvalidHarvestDateFormat)
	}

	process := lot.ParseProcess(req.Process)
	if !process.IsValid() {
		return lot.Default(), exception.New(exception.CodeLotInvalidProcess)
	}

	return lot.Draft(farmID, req.Code, req.Variety, process, harvestDate), nil
}

func (Assembler) ToDTO(l *lot.Lot) response.Lot {
	return lotmapper.ToResponse(l)
}
