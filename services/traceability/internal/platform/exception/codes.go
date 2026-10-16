package exception

type Code string

const (
	CodeInternal           Code = "INTERNAL"
	CodeInvalidRequestBody Code = "INVALID_REQUEST_BODY"
	CodeUnauthorized       Code = "UNAUTHORIZED"
	CodeForbidden          Code = "FORBIDDEN"
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

	CodeActorInvalidUserID Code = "TRACEABILITY_ACTOR_INVALID_USER_ID"
	CodeActorInvalidName   Code = "TRACEABILITY_ACTOR_INVALID_NAME"
	CodeActorInvalidType   Code = "TRACEABILITY_ACTOR_INVALID_TYPE"
	CodeActorNotFound      Code = "TRACEABILITY_ACTOR_NOT_FOUND"

	CodeFarmInvalidProducer Code = "TRACEABILITY_FARM_INVALID_PRODUCER"
	CodeFarmInvalidName     Code = "TRACEABILITY_FARM_INVALID_NAME"
	CodeFarmInvalidAltitude Code = "TRACEABILITY_FARM_INVALID_ALTITUDE"
	CodeFarmNotFound        Code = "TRACEABILITY_FARM_NOT_FOUND"
)

var kindByCode = map[Code]Kind{
	CodeInternal:           KindInternal,
	CodeInvalidRequestBody: KindValidation,
	CodeUnauthorized:       KindUnauthorized,
	CodeForbidden:          KindForbidden,

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

	CodeActorInvalidUserID: KindValidation,
	CodeActorInvalidName:   KindValidation,
	CodeActorInvalidType:   KindValidation,
	CodeActorNotFound:      KindNotFound,

	CodeFarmInvalidProducer: KindValidation,
	CodeFarmInvalidName:     KindValidation,
	CodeFarmInvalidAltitude: KindValidation,
	CodeFarmNotFound:        KindNotFound,
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
