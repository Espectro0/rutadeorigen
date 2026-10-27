package controllers

import (
	"errors"
	"net/http"

	"rutadeorigen/lib/httpserver/middleware"
	"rutadeorigen/lib/session"
	"rutadeorigen/qr/internal/platform/exception"
)

func RejectSession(errs *ErrorResponder) middleware.RejectFunc {
	return func(w http.ResponseWriter, r *http.Request, err error) {
		switch {
		case errors.Is(err, session.ErrUnauthenticated):
			errs.Write(w, r, exception.New(exception.CodeUnauthorized))
		case errors.Is(err, session.ErrAuthUnavailable):
			errs.Write(w, r, exception.New(exception.CodeAuthUnavailable).WithCause(err))
		default:
			errs.Write(w, r, exception.New(exception.CodeInternal).WithTechnical("authenticate request").WithCause(err))
		}
	}
}
