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
	exception.KindUnavailable:  http.StatusServiceUnavailable,
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
		ex = exception.New(exception.CodeInternal).
			WithTechnical("unclassified error").
			WithCause(err)
	}

	status, known := statusByKind[ex.Kind()]
	if !known {
		status = http.StatusInternalServerError
	}

	code, kind, params := ex.Code(), ex.Kind(), ex.Params()
	if status == http.StatusInternalServerError {
		code, kind, params = exception.CodeInternal, exception.KindInternal, nil
	}

	e.logFailure(ctx, ex, status)

	writeJSON(w, e.log, status, errorResponse{
		Code:          string(code),
		Error:         e.message(ctx, code, params),
		Kind:          string(kind),
		CorrelationID: requestcontext.CorrelationID(ctx),
	})
}

func (e *ErrorResponder) logFailure(ctx context.Context, ex *exception.Exception, status int) {
	attrs := []any{"code", ex.Code(), "technical", ex.TechnicalMessage()}

	if status >= http.StatusInternalServerError {
		e.log.ErrorContext(ctx, "request failed", attrs...)
	} else {
		e.log.WarnContext(ctx, "request rejected", attrs...)
	}

	if cause := ex.Unwrap(); cause != nil {
		e.log.DebugContext(ctx, "request failure cause", "code", ex.Code(), "cause", cause)
	}
}

func (e *ErrorResponder) message(ctx context.Context, code exception.Code, params map[string]any) string {
	msg, err := e.messages.Resolve(string(code), params)
	if err != nil {
		e.log.ErrorContext(ctx, "cannot resolve error message", "code", code, "err", err)
	}

	return msg
}
