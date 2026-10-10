package controllers

import (
	"log/slog"
	"net/http"
	"rutadeorigen/lib/requestcontext"
	"rutadeorigen/traceability/internal/platform/exception"
	"rutadeorigen/traceability/internal/platform/helper"
)

const ActorIDHeader = "X-Actor-ID"

var errMissingActor = exception.NewUnauthorized(exception.LayerInfrastructure,
	"Debes identificarte para realizar esta operación", "missing or invalid X-Actor-ID header")

func RequireActor(log *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		actorID := helper.ParseUUID(r.Header.Get(ActorIDHeader))
		if helper.IsDefaultUUID(actorID) {
			writeError(w, r, log, errMissingActor)
			return
		}

		ctx := requestcontext.WithActorID(r.Context(), actorID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
