package controllers

import (
	"log/slog"
	"net/http"

	"rutadeorigen/lib/requestcontext"
	"rutadeorigen/traceability/internal/platform/exception"
)

const genericErrorMessage = "Ocurrió un error inesperado, intenta de nuevo más tarde"

type errorResponse struct {
	Error         string `json:"error"`
	Kind          string `json:"kind"`
	CorrelationID string `json:"correlation_id,omitempty"`
}

var statusByKind = map[exception.Kind]int{
	exception.KindValidation:   http.StatusBadRequest,
	exception.KindNotFound:     http.StatusNotFound,
	exception.KindForbidden:    http.StatusForbidden,
	exception.KindConflict:     http.StatusConflict,
	exception.KindInternal:     http.StatusInternalServerError,
	exception.KindUnauthorized: http.StatusUnauthorized,
}

func writeError(w http.ResponseWriter, r *http.Request, log *slog.Logger, err error) {
	correlationID := requestcontext.CorrelationID(r.Context())

	ex, ok := exception.From(err)
	if !ok {
		ex = exception.NewInternal(exception.LayerInfrastructure, genericErrorMessage, "unclassified error", err)
	}

	status, known := statusByKind[ex.Kind()]
	if !known {
		status = http.StatusInternalServerError
	}

	body := errorResponse{
		Error:         ex.UserMessage(),
		Kind:          string(ex.Kind()),
		CorrelationID: correlationID,
	}

	if status == http.StatusInternalServerError {
		log.ErrorContext(r.Context(), "request failed",
			"layer", ex.Layer(),
			"err", ex.Error(),
		)
		body.Error = genericErrorMessage
		body.Kind = string(exception.KindInternal)
	}

	writeJSON(w, log, status, body)
}
