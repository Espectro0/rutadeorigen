package controllers

import (
	"context"
	"log/slog"
	"net/http"

	"rutadeorigen/lib/requestcontext"
	"rutadeorigen/traceability/internal/platform/exception"
)

type MessageResolver interface {
	Resolve(code string, params map[string]any) (string, error)
}

type errorResponse struct {
	Code          string `json:"code,omitempty"`
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

type ErrorResponder struct {
	log      *slog.Logger
	messages MessageResolver
}

func NewErrorResponder(log *slog.Logger, messages MessageResolver) *ErrorResponder {
	return &ErrorResponder{log: log, messages: messages}
}

func (e *ErrorResponder) Write(w http.ResponseWriter, r *http.Request, err error) {
	ctx := r.Context()

	ex, ok := exception.From(err)
	if !ok {
		ex = exception.NewInternal(exception.LayerInfrastructure, "", "unclassified error", err)
	}

	status, known := statusByKind[ex.Kind()]
	if !known {
		status = http.StatusInternalServerError
	}

	body := errorResponse{
		Kind:          string(ex.Kind()),
		CorrelationID: requestcontext.CorrelationID(ctx),
	}

	switch {
	case status == http.StatusInternalServerError:
		e.log.ErrorContext(ctx, "request failed",
			"layer", ex.Layer(),
			"err", ex.Error(),
		)
		body.Code = string(exception.CodeInternal)
		body.Kind = string(exception.KindInternal)
		body.Error = e.message(ctx, exception.CodeInternal, nil)
	case ex.Code() != "":
		body.Code = string(ex.Code())
		body.Error = e.message(ctx, ex.Code(), ex.Params())
	default:
		body.Error = ex.UserMessage()
	}

	writeJSON(w, e.log, status, body)
}

func (e *ErrorResponder) message(ctx context.Context, code exception.Code, params map[string]any) string {
	msg, err := e.messages.Resolve(string(code), params)
	if err != nil {
		e.log.ErrorContext(ctx, "cannot resolve error message", "code", code, "err", err)
	}

	return msg
}
