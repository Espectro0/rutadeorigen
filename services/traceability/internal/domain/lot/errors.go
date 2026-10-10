package lot

import "rutadeorigen/traceability/internal/platform/exception"

var (
	ErrInvalidCode = exception.NewValidation(exception.LayerDomain,
		"The lot code must have between 3 and 30 characters", "")
	ErrInvalidVariety = exception.NewValidation(exception.LayerDomain,
		"The coffee variety is required", "")
	ErrInvalidProcess = exception.NewValidation(exception.LayerDomain,
		"The process must be washed, natural or honey", "")
	ErrInvalidHarvestDate = exception.NewValidation(exception.LayerDomain,
		"The harvest date is required and cannot be in the future", "")
	ErrInvalidFarm = exception.NewValidation(exception.LayerDomain,
		"The lot must belong to a farm", "")
	ErrInvalidCustodian = exception.NewValidation(exception.LayerDomain,
		"The lot must have a custodian", "")
	ErrVoided = exception.NewConflict(exception.LayerDomain,
		"The lot is voided and cannot be changed", "")
	ErrEmptyVoidReason = exception.NewValidation(exception.LayerDomain,
		"A reason is required to void a lot", "")
	ErrHasEvents = exception.NewConflict(exception.LayerDomain,
		"The lot already has registered events and cannot be changed or deleted", "")
	ErrStageOutOfOrder = exception.NewConflict(exception.LayerDomain,
		"Stages must be registered in order: farm, roasting and then packaging", "")
)
