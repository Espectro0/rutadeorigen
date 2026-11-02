package exception

type Code string

const (
	CodeInternal           Code = "INTERNAL"
	CodeInvalidRequestBody Code = "INVALID_REQUEST_BODY"
	CodeUnauthorized       Code = "UNAUTHORIZED"
	CodeForbidden          Code = "FORBIDDEN"
	CodeInvalidCursor      Code = "INVALID_CURSOR"
	CodeInvalidLimit       Code = "INVALID_LIMIT"
	CodeAuthUnavailable    Code = "AUTH_UNAVAILABLE"
)

const (
	CodeEventInvalidPayload     Code = "NOTIFICATIONS_EVENT_INVALID_PAYLOAD"
	CodeInvalidRecipient        Code = "NOTIFICATIONS_INVALID_RECIPIENT"
	CodeInvalidStatusTransition Code = "NOTIFICATIONS_INVALID_STATUS_TRANSITION"

	CodeEmailRejected     Code = "NOTIFICATIONS_EMAIL_REJECTED"
	CodeEmailUnavailable  Code = "NOTIFICATIONS_EMAIL_UNAVAILABLE"
	CodeEmailRenderFailed Code = "NOTIFICATIONS_EMAIL_RENDER_FAILED"
)

var kindByCode = map[Code]Kind{
	CodeInternal:           KindInternal,
	CodeInvalidRequestBody: KindValidation,
	CodeUnauthorized:       KindUnauthorized,
	CodeForbidden:          KindForbidden,
	CodeInvalidCursor:      KindValidation,
	CodeInvalidLimit:       KindValidation,
	CodeAuthUnavailable:    KindUnavailable,

	CodeEventInvalidPayload:     KindValidation,
	CodeInvalidRecipient:        KindValidation,
	CodeInvalidStatusTransition: KindConflict,

	CodeEmailRejected:     KindValidation,
	CodeEmailUnavailable:  KindUnavailable,
	CodeEmailRenderFailed: KindInternal,
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
