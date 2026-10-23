package postgres

import "rutadeorigen/qr/internal/platform/exception"

func internalError(technicalMessage string, cause error) error {
	return exception.New(exception.CodeInternal).
		WithTechnical(technicalMessage).
		WithCause(cause)
}
