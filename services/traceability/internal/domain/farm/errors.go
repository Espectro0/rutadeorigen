package farm

import "rutadeorigen/traceability/internal/platform/exception"

var (
	ErrInvalidProducer = exception.NewValidation(exception.LayerDomain,
		"The farm must belong to a producer", "")
	ErrInvalidName = exception.NewValidation(exception.LayerDomain,
		"The farm name must have between 3 and 100 characters", "")
	ErrInvalidAltitude = exception.NewValidation(exception.LayerDomain,
		"The altitude must be between 0 and 3000 meters above sea level", "")
)
