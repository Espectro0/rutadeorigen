package actor

import "rutadeorigen/traceability/internal/platform/exception"

var (
	ErrInvalidUserID = exception.NewValidation(exception.LayerDomain,
		"The actor must be linked to a user", "")
	ErrInvalidName = exception.NewValidation(exception.LayerDomain,
		"The actor name must have between 3 and 100 characters", "")
	ErrInvalidType = exception.NewValidation(exception.LayerDomain,
		"The actor type must be organization, producer, roaster or brand", "")
)
