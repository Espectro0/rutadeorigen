package usecases

import "rutadeorigen/traceability/internal/platform/exception"

var (
	ErrActorNotFound = exception.NewNotFound(exception.LayerApplication,
		"No se encontró ningún actor", "")

	ErrFarmNotFound = exception.NewNotFound(exception.LayerApplication,
		"No se encontró ninguna granja", "")

	ErrNotAllowed = exception.NewForbidden(exception.LayerApplication,
		"Usuario no está autorizado para realizar esta acción", "")

	ErrLotCodeAlreadyExists = exception.NewConflict(exception.LayerApplication,
		"Lote ya existe", "")
)
