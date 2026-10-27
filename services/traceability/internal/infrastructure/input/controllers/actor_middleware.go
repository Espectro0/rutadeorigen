package controllers

import (
	"net/http"

	"rutadeorigen/lib/requestcontext"
	"rutadeorigen/traceability/internal/application/input/dto/request"
	"rutadeorigen/traceability/internal/application/input/ports"
	"rutadeorigen/traceability/internal/platform/exception"
	"rutadeorigen/traceability/internal/platform/helper"
)

const ActorIDHeader = "X-Actor-ID"

func RequireActor(resolver ports.ResolveActorPort, errs *ErrorResponder, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		found, err := resolver.Execute(r.Context(), request.ResolveActor{})
		if err != nil {
			errs.Write(w, r, err)
			return
		}

		ctx := requestcontext.WithActorID(r.Context(), found.ID)
		next.ServeHTTP(w, r.WithContext(ctx))
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
