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
	CodeEventInvalidPayload Code = "PROCESSOR_EVENT_INVALID_PAYLOAD"
	CodeEvidenceKeyMismatch Code = "PROCESSOR_EVIDENCE_KEY_MISMATCH"

	CodeFileMissing      Code = "PROCESSOR_FILE_MISSING"
	CodeFileTooLarge     Code = "PROCESSOR_FILE_TOO_LARGE"
	CodeFileSizeMismatch Code = "PROCESSOR_FILE_SIZE_MISMATCH"

	CodeImageUnsupportedFormat Code = "PROCESSOR_IMAGE_UNSUPPORTED_FORMAT"
	CodeImageCorrupt           Code = "PROCESSOR_IMAGE_CORRUPT"
	CodeImageInvalidDimensions Code = "PROCESSOR_IMAGE_INVALID_DIMENSIONS"
	CodeImageTooManyPixels     Code = "PROCESSOR_IMAGE_TOO_MANY_PIXELS"
	CodeImageProcessingFailed  Code = "PROCESSOR_IMAGE_PROCESSING_FAILED"

	CodeStorageUnavailable Code = "PROCESSOR_STORAGE_UNAVAILABLE"
)

var kindByCode = map[Code]Kind{
	CodeInternal:           KindInternal,
	CodeInvalidRequestBody: KindValidation,
	CodeUnauthorized:       KindUnauthorized,
	CodeForbidden:          KindForbidden,
	CodeInvalidCursor:      KindValidation,
	CodeInvalidLimit:       KindValidation,
	CodeAuthUnavailable:    KindUnavailable,

	CodeEventInvalidPayload: KindValidation,
	CodeEvidenceKeyMismatch: KindValidation,

	CodeFileMissing:      KindValidation,
	CodeFileTooLarge:     KindValidation,
	CodeFileSizeMismatch: KindValidation,

	CodeImageUnsupportedFormat: KindValidation,
	CodeImageCorrupt:           KindValidation,
	CodeImageInvalidDimensions: KindValidation,
	CodeImageTooManyPixels:     KindValidation,
	CodeImageProcessingFailed:  KindInternal,

	CodeStorageUnavailable: KindUnavailable,
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
