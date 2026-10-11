package createlot

import "rutadeorigen/traceability/internal/platform/exception"

var (
	ErrInvalidFarmID = exception.NewValidation(exception.LayerApplication,
		"El identificador de la finca no es válido", "farm_id is not a valid UUID")
	ErrInvalidHarvestDate = exception.NewValidation(exception.LayerApplication,
		"La fecha de cosecha debe tener el formato AAAA-MM-DD", "harvest_date is not a valid date")
	ErrInvalidProcess = exception.NewValidation(exception.LayerApplication,
		"El proceso debe ser washed, natural o honey", "process is not a known value")
)
