package registerstage

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strconv"
	"strings"

	"rutadeorigen/traceability/internal/application/input/adapters/lotmapper"
	"rutadeorigen/traceability/internal/application/input/dto/request"
	"rutadeorigen/traceability/internal/application/input/dto/response"
	registerstageuc "rutadeorigen/traceability/internal/application/usecases/registerstage"
	"rutadeorigen/traceability/internal/domain/lot"
	"rutadeorigen/traceability/internal/platform/exception"
	"rutadeorigen/traceability/internal/platform/helper"
)

type Assembler struct{}

func (Assembler) ToCommand(req request.RegisterStage) (registerstageuc.Command, error) {
	lotID := helper.ParseUUID(req.LotID)
	if helper.IsDefaultUUID(lotID) {
		return registerstageuc.Command{}, exception.New(exception.CodeLotNotFound)
	}

	version, err := parseIfMatch(req.IfMatch)
	if err != nil {
		return registerstageuc.Command{}, err
	}

	stage := lot.ParseStage(req.Stage)
	if !stage.IsValid() || stage == lot.StageNone {
		return registerstageuc.Command{}, exception.New(exception.CodeLotInvalidStage)
	}

	eventDate := helper.ParseDate(req.EventDate)
	if helper.IsDefaultDate(eventDate) {
		return registerstageuc.Command{}, exception.New(exception.CodeLotInvalidEventDate)
	}

	details, err := decodeDetails(stage, req.Details)
	if err != nil {
		return registerstageuc.Command{}, err
	}

	return registerstageuc.Command{
		LotID:           lotID,
		ExpectedVersion: version,
		Stage:           stage,
		EventDate:       eventDate,
		Note:            req.Note,
		Details:         details,
	}, nil
}

func (Assembler) ToDTO(res registerstageuc.Result) response.RegisteredStage {
	return response.RegisteredStage{
		Event:      lotmapper.ToEventResponse(res.Event),
		LotVersion: res.LotVersion,
	}
}

func parseIfMatch(header string) (int, error) {
	value := strings.TrimPrefix(helper.ApplyTrim(header), "W/")
	value = strings.Trim(value, `"`)

	version, err := strconv.Atoi(value)
	if err != nil || version < 1 {
		return 0, exception.New(exception.CodeLotInvalidIfMatch)
	}
	return version, nil
}

func decodeDetails(stage lot.Stage, raw json.RawMessage) (lot.StageDetails, error) {
	switch stage {
	case lot.StageFarm:
		d, err := decode[lot.FarmDetails](raw)
		if err != nil {
			return nil, err
		}
		d.DryingMethod = lot.ParseDryingMethod(string(d.DryingMethod))
		return d, nil
	case lot.StageRoasting:
		d, err := decode[lot.RoastingDetails](raw)
		if err != nil {
			return nil, err
		}
		d.RoastLevel = lot.ParseRoastLevel(string(d.RoastLevel))
		return d, nil
	case lot.StagePackaging:
		d, err := decode[lot.PackagingDetails](raw)
		if err != nil {
			return nil, err
		}
		d.Grind = lot.ParseGrind(string(d.Grind))
		return d, nil
	default:
		return nil, exception.New(exception.CodeLotInvalidStage)
	}
}

func decode[T lot.StageDetails](raw json.RawMessage) (T, error) {
	var details T

	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return details, exception.New(exception.CodeStageInvalidDetails)
	}

	decoder := json.NewDecoder(bytes.NewReader(trimmed))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&details); err != nil {
		return details, exception.New(exception.CodeStageInvalidDetails).WithCause(err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return details, exception.New(exception.CodeStageInvalidDetails)
	}
	return details, nil
}
