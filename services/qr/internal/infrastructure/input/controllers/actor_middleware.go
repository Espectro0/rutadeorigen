package controllers

import (
	"net/http"

	"rutadeorigen/qr/internal/platform/exception"

	"rutadeorigen/lib/requestcontext"
	"rutadeorigen/qr/internal/platform/helper"

	"github.com/google/uuid"
)

const ActorIDHeader = "X-Actor-ID"

func ActorFromSession(errs *ErrorResponder, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := requestcontext.UserID(r.Context())
		if userID == "" {
			errs.Write(w, r, exception.New(exception.CodeUnauthorized))
			return
		}
		id := uuid.NewSHA1(uuid.NameSpaceURL, []byte(userID))
		next.ServeHTTP(w, r.WithContext(requestcontext.WithActorID(r.Context(), id)))
	})
}

func RequireActorHeader(errs *ErrorResponder, next http.Handler) http.Handler {
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
