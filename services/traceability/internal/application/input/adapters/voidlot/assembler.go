package voidlot

import (
	"rutadeorigen/traceability/internal/application/input/adapters/ifmatch"
	"rutadeorigen/traceability/internal/application/input/dto/request"
	voidlotuc "rutadeorigen/traceability/internal/application/usecases/voidlot"
	"rutadeorigen/traceability/internal/platform/exception"
	"rutadeorigen/traceability/internal/platform/helper"
)

type Assembler struct{}

func (Assembler) ToCommand(req request.VoidLot) (voidlotuc.Command, error) {
	lotID := helper.ParseUUID(req.LotID)
	if helper.IsDefaultUUID(lotID) {
		return voidlotuc.Command{}, exception.New(exception.CodeLotNotFound)
	}

	version, err := ifmatch.Parse(req.IfMatch)
	if err != nil {
		return voidlotuc.Command{}, err
	}

	return voidlotuc.Command{LotID: lotID, ExpectedVersion: version, Reason: req.Reason}, nil
}
