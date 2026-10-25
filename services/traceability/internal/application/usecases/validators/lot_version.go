package validators

import (
	"rutadeorigen/traceability/internal/domain/lot"
	"rutadeorigen/traceability/internal/platform/exception"
)

func LotVersion(l *lot.Lot, expected int) error {
	if l.IsDefault() {
		return exception.New(exception.CodeLotNotFound)
	}
	if l.Version() != expected {
		return exception.New(exception.CodeLotVersionConflict)
	}
	return nil
}
