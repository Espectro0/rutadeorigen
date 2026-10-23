package controllers

import (
	"net/http"

	"rutadeorigen/qr/internal/platform/exception"

	"rutadeorigen/lib/requestcontext"
	"rutadeorigen/qr/internal/platform/helper"
)

const ActorIDHeader = "X-Actor-ID"

func RequireActor(errs *ErrorResponder, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		actorID := helper.ParseUUID(r.Header.Get(ActorIDHeader))
		if helper.IsDefaultUUID(actorID) {
			errs.Write(w, r, exception.New(exception.CodeUnauthorized))
			return
		}

		ctx := requestcontext.WithActorID(r.Context(), actorID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
