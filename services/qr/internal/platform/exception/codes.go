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
	CodeLotViewInvalidID         Code = "QR_LOT_VIEW_INVALID_ID"
	CodeLotViewInvalidCode       Code = "QR_LOT_VIEW_INVALID_CODE"
	CodeLotViewInvalidVariety    Code = "QR_LOT_VIEW_INVALID_VARIETY"
	CodeLotViewInvalidProcess    Code = "QR_LOT_VIEW_INVALID_PROCESS"
	CodeLotViewInvalidHarvest    Code = "QR_LOT_VIEW_INVALID_HARVEST_DATE"
	CodeLotViewInvalidFarm       Code = "QR_LOT_VIEW_INVALID_FARM"
	CodeLotViewInvalidProducer   Code = "QR_LOT_VIEW_INVALID_PRODUCER"
	CodeLotViewInvalidCustodian  Code = "QR_LOT_VIEW_INVALID_CUSTODIAN"
	CodeLotViewInvalidStage      Code = "QR_LOT_VIEW_INVALID_STAGE"
	CodeLotViewInvalidStatus     Code = "QR_LOT_VIEW_INVALID_STATUS"
	CodeLotViewInvalidVersion    Code = "QR_LOT_VIEW_INVALID_VERSION"
	CodeLotViewNotFound          Code = "QR_LOT_VIEW_NOT_FOUND"
	CodeLotViewImageNotAvailable Code = "QR_LOT_VIEW_IMAGE_NOT_AVAILABLE"
	CodeLotViewVoided            Code = "QR_LOT_VIEW_VOIDED"

	CodeQRCodeInvalidContent         Code = "QR_CODE_INVALID_CONTENT"
	CodeQRCodeInvalidSize            Code = "QR_CODE_INVALID_SIZE"
	CodeQRCodeInvalidFormat          Code = "QR_CODE_INVALID_FORMAT"
	CodeQRCodeInvalidErrorCorrection Code = "QR_CODE_INVALID_ERROR_CORRECTION"

	CodeRedirectInvalidRule Code = "QR_REDIRECT_INVALID_RULE"
	CodeRedirectNotFound    Code = "QR_REDIRECT_NOT_FOUND"

	CodeEventInvalidPayload Code = "QR_EVENT_INVALID_PAYLOAD"
)

var kindByCode = map[Code]Kind{
	CodeInternal:           KindInternal,
	CodeInvalidRequestBody: KindValidation,
	CodeUnauthorized:       KindUnauthorized,
	CodeForbidden:          KindForbidden,
	CodeInvalidCursor:      KindValidation,
	CodeInvalidLimit:       KindValidation,

	CodeLotViewInvalidID:         KindValidation,
	CodeLotViewInvalidCode:       KindValidation,
	CodeLotViewInvalidVariety:    KindValidation,
	CodeLotViewInvalidProcess:    KindValidation,
	CodeLotViewInvalidHarvest:    KindValidation,
	CodeLotViewInvalidFarm:       KindValidation,
	CodeLotViewInvalidProducer:   KindValidation,
	CodeLotViewInvalidCustodian:  KindValidation,
	CodeLotViewInvalidStage:      KindValidation,
	CodeLotViewInvalidStatus:     KindValidation,
	CodeLotViewInvalidVersion:    KindValidation,
	CodeLotViewNotFound:          KindNotFound,
	CodeLotViewImageNotAvailable: KindConflict,
	CodeLotViewVoided:            KindConflict,

	CodeQRCodeInvalidContent:         KindValidation,
	CodeQRCodeInvalidSize:            KindValidation,
	CodeQRCodeInvalidFormat:          KindValidation,
	CodeQRCodeInvalidErrorCorrection: KindValidation,

	CodeRedirectInvalidRule: KindValidation,
	CodeRedirectNotFound:    KindNotFound,

	CodeEventInvalidPayload: KindValidation,
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
