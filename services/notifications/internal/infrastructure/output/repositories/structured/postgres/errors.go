package postgres

import "rutadeorigen/notifications/internal/platform/exception"

func internalError(technicalMessage string, cause error) error {
	return exception.New(exception.CodeInternal).
		WithTechnical(technicalMessage).
		WithCause(cause)
}
