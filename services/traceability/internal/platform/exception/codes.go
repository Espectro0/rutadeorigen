package exception

type Code string

const (
	CodeInternal           Code = "INTERNAL"
	CodeInvalidRequestBody Code = "INVALID_REQUEST_BODY"
	CodeUnauthorized       Code = "UNAUTHORIZED"
	CodeForbidden          Code = "FORBIDDEN"
	CodeInvalidCursor      Code = "INVALID_CURSOR"
	CodeInvalidLimit       Code = "INVALID_LIMIT"
)

const (
	CodeLotInvalidCode              Code = "TRACEABILITY_LOT_INVALID_CODE"
	CodeLotInvalidVariety           Code = "TRACEABILITY_LOT_INVALID_VARIETY"
	CodeLotInvalidProcess           Code = "TRACEABILITY_LOT_INVALID_PROCESS"
	CodeLotInvalidHarvestDate       Code = "TRACEABILITY_LOT_INVALID_HARVEST_DATE"
	CodeLotInvalidHarvestDateFormat Code = "TRACEABILITY_LOT_INVALID_HARVEST_DATE_FORMAT"
	CodeLotInvalidFarm              Code = "TRACEABILITY_LOT_INVALID_FARM"
	CodeLotInvalidFarmID            Code = "TRACEABILITY_LOT_INVALID_FARM_ID"
	CodeLotInvalidCustodian         Code = "TRACEABILITY_LOT_INVALID_CUSTODIAN"
	CodeLotEmptyVoidReason          Code = "TRACEABILITY_LOT_EMPTY_VOID_REASON"
	CodeLotVoided                   Code = "TRACEABILITY_LOT_VOIDED"
	CodeLotHasEvents                Code = "TRACEABILITY_LOT_HAS_EVENTS"
	CodeLotStageOutOfOrder          Code = "TRACEABILITY_LOT_STAGE_OUT_OF_ORDER"
	CodeLotCodeAlreadyExists        Code = "TRACEABILITY_LOT_CODE_ALREADY_EXISTS"
	CodeLotNotFound                 Code = "TRACEABILITY_LOT_NOT_FOUND"
	CodeLotInvalidCustodianID       Code = "TRACEABILITY_LOT_INVALID_CUSTODIAN_ID"
	CodeLotVersionConflict          Code = "TRACEABILITY_LOT_VERSION_CONFLICT"
	CodeLotInvalidIfMatch           Code = "TRACEABILITY_LOT_INVALID_IF_MATCH"
	CodeLotInvalidStage             Code = "TRACEABILITY_LOT_INVALID_STAGE"
	CodeLotInvalidEventDate         Code = "TRACEABILITY_LOT_INVALID_EVENT_DATE"
	CodeLotInvalidNote              Code = "TRACEABILITY_LOT_INVALID_NOTE"
	CodeStageInvalidDetails         Code = "TRACEABILITY_STAGE_INVALID_DETAILS"

	CodeActorInvalidUserID Code = "TRACEABILITY_ACTOR_INVALID_USER_ID"
	CodeActorInvalidName   Code = "TRACEABILITY_ACTOR_INVALID_NAME"
	CodeActorInvalidType   Code = "TRACEABILITY_ACTOR_INVALID_TYPE"
	CodeActorNotFound      Code = "TRACEABILITY_ACTOR_NOT_FOUND"

	CodeFarmInvalidProducer Code = "TRACEABILITY_FARM_INVALID_PRODUCER"
	CodeFarmInvalidName     Code = "TRACEABILITY_FARM_INVALID_NAME"
	CodeFarmInvalidAltitude Code = "TRACEABILITY_FARM_INVALID_ALTITUDE"
	CodeFarmNotFound        Code = "TRACEABILITY_FARM_NOT_FOUND"

	CodeStageInvalidDryingMethod    Code = "TRACEABILITY_STAGE_INVALID_DRYING_METHOD"
	CodeStageInvalidDryingDays      Code = "TRACEABILITY_STAGE_INVALID_DRYING_DAYS"
	CodeStageInvalidMoisture        Code = "TRACEABILITY_STAGE_INVALID_MOISTURE"
	CodeStageInvalidParchmentWeight Code = "TRACEABILITY_STAGE_INVALID_PARCHMENT_WEIGHT"

	CodeStageInvalidRoastLevel    Code = "TRACEABILITY_STAGE_INVALID_ROAST_LEVEL"
	CodeStageInvalidRoastDuration Code = "TRACEABILITY_STAGE_INVALID_ROAST_DURATION"
	CodeStageInvalidGreenWeight   Code = "TRACEABILITY_STAGE_INVALID_GREEN_WEIGHT"
	CodeStageInvalidRoastedWeight Code = "TRACEABILITY_STAGE_INVALID_ROASTED_WEIGHT"

	CodeStageInvalidGrind         Code = "TRACEABILITY_STAGE_INVALID_GRIND"
	CodeStageInvalidPackageWeight Code = "TRACEABILITY_STAGE_INVALID_PACKAGE_WEIGHT"
	CodeStageInvalidUnit          Code = "TRACEABILITY_STAGE_INVALID_UNIT"
	CodeStageInvalidBestBefore    Code = "TRACEABILITY_STAGE_INVALID_BEST_BEFORE"
)

var kindByCode = map[Code]Kind{
	CodeInternal:           KindInternal,
	CodeInvalidRequestBody: KindValidation,
	CodeUnauthorized:       KindUnauthorized,
	CodeForbidden:          KindForbidden,
	CodeInvalidCursor:      KindValidation,
	CodeInvalidLimit:       KindValidation,

	CodeLotInvalidCode:              KindValidation,
	CodeLotInvalidVariety:           KindValidation,
	CodeLotInvalidProcess:           KindValidation,
	CodeLotInvalidHarvestDate:       KindValidation,
	CodeLotInvalidHarvestDateFormat: KindValidation,
	CodeLotInvalidFarm:              KindValidation,
	CodeLotInvalidFarmID:            KindValidation,
	CodeLotInvalidCustodian:         KindValidation,
	CodeLotEmptyVoidReason:          KindValidation,
	CodeLotVoided:                   KindConflict,
	CodeLotHasEvents:                KindConflict,
	CodeLotStageOutOfOrder:          KindConflict,
	CodeLotCodeAlreadyExists:        KindConflict,
	CodeLotNotFound:                 KindNotFound,
	CodeLotInvalidCustodianID:       KindValidation,
	CodeLotVersionConflict:          KindConflict,
	CodeLotInvalidIfMatch:           KindValidation,
	CodeLotInvalidStage:             KindValidation,
	CodeLotInvalidEventDate:         KindValidation,
	CodeLotInvalidNote:              KindValidation,
	CodeStageInvalidDetails:         KindValidation,

	CodeActorInvalidUserID: KindValidation,
	CodeActorInvalidName:   KindValidation,
	CodeActorInvalidType:   KindValidation,
	CodeActorNotFound:      KindNotFound,

	CodeFarmInvalidProducer: KindValidation,
	CodeFarmInvalidName:     KindValidation,
	CodeFarmInvalidAltitude: KindValidation,
	CodeFarmNotFound:        KindNotFound,

	CodeStageInvalidDryingMethod:    KindValidation,
	CodeStageInvalidDryingDays:      KindValidation,
	CodeStageInvalidMoisture:        KindValidation,
	CodeStageInvalidParchmentWeight: KindValidation,

	CodeStageInvalidRoastLevel:    KindValidation,
	CodeStageInvalidRoastDuration: KindValidation,
	CodeStageInvalidGreenWeight:   KindValidation,
	CodeStageInvalidRoastedWeight: KindValidation,

	CodeStageInvalidGrind:         KindValidation,
	CodeStageInvalidPackageWeight: KindValidation,
	CodeStageInvalidUnit:          KindValidation,
	CodeStageInvalidBestBefore:    KindValidation,
}

func KindOf(c Code) (Kind, bool) {
	k, ok := kindByCode[c]
	return k, ok
}

func AllCodes() []Code {
	codes := make([]Code, 0, len(kindByCode))
	for c := range kindByCode {
		codes = append(codes, c)
	}
	return codes
}
