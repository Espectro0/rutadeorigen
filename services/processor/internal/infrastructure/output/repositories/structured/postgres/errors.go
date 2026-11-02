package postgres

import "rutadeorigen/processor/internal/platform/exception"

func internalError(technicalMessage string, cause error) error {
	return exception.New(exception.CodeInternal).
		WithTechnical(technicalMessage).
		WithCause(cause)
}
