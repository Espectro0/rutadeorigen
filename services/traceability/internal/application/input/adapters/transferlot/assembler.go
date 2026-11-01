package transferlot

import (
	"rutadeorigen/traceability/internal/application/input/adapters/ifmatch"
	"rutadeorigen/traceability/internal/application/input/adapters/lotmapper"
	"rutadeorigen/traceability/internal/application/input/dto/request"
	"rutadeorigen/traceability/internal/application/input/dto/response"
	transferlotuc "rutadeorigen/traceability/internal/application/usecases/transferlot"
	"rutadeorigen/traceability/internal/platform/exception"
	"rutadeorigen/traceability/internal/platform/helper"
)

type Assembler struct{}

func (Assembler) ToCommand(req request.TransferLot) (transferlotuc.Command, error) {
	lotID := helper.ParseUUID(req.LotID)
	if helper.IsDefaultUUID(lotID) {
		return transferlotuc.Command{}, exception.New(exception.CodeLotNotFound)
	}

	version, err := ifmatch.Parse(req.IfMatch)
	if err != nil {
		return transferlotuc.Command{}, err
	}

	targetID := helper.ParseUUID(req.TargetActorID)
	if helper.IsDefaultUUID(targetID) {
		return transferlotuc.Command{}, exception.New(exception.CodeActorNotFound)
	}

	return transferlotuc.Command{
		LotID:           lotID,
		ExpectedVersion: version,
		TargetActorID:   targetID,
		Note:            req.Note,
	}, nil
}

func (Assembler) ToDTO(res transferlotuc.Result) response.TransferredLot {
	return response.TransferredLot{
		Event:      lotmapper.ToEventResponse(res.Event),
		LotVersion: res.LotVersion,
	}
}
