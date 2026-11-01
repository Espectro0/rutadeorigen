package controllers

import (
	"net/http"

	"rutadeorigen/lib/requestcontext"
	"rutadeorigen/qr/internal/application/input/dto/request"
	"rutadeorigen/qr/internal/application/input/ports"
	"rutadeorigen/qr/internal/platform/exception"
	"rutadeorigen/qr/internal/platform/helper"
)

const ActorIDHeader = "X-Actor-ID"

// ActorFromSession resolves the actor of the authenticated user from the
// projection fed by actor.registered. A user the projection does not know yet
// is rejected: the actor id and role are never derived locally.
func ActorFromSession(errs *ErrorResponder, resolve ports.ResolveActorPort, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := requestcontext.UserID(r.Context())
		if userID == "" {
			errs.Write(w, r, exception.New(exception.CodeUnauthorized))
			return
		}

		actor, err := resolve.Execute(r.Context(), request.ResolveActor{UserID: userID})
		if err != nil {
			errs.Write(w, r, err)
			return
		}

		ctx := requestcontext.WithActorID(r.Context(), actor.ActorID)
		ctx = requestcontext.WithActorRole(ctx, actor.Role)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireActorHeader trusts the actor id of the header (development only).
// When a resolver is given, the role is taken from the projection if the actor
// is already known there.
func RequireActorHeader(errs *ErrorResponder, resolve ports.ResolveActorPort, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		actorID := helper.ParseUUID(r.Header.Get(ActorIDHeader))
		if helper.IsDefaultUUID(actorID) {
			errs.Write(w, r, exception.New(exception.CodeUnauthorized))
			return
		}

		ctx := requestcontext.WithActorID(r.Context(), actorID)
		if resolve != nil {
			actor, err := resolve.Execute(r.Context(), request.ResolveActor{ActorID: actorID.String()})
			switch {
			case err == nil:
				ctx = requestcontext.WithActorRole(ctx, actor.Role)
			case !exception.HasCode(err, exception.CodeActorViewNotFound):
				errs.Write(w, r, err)
				return
			}
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
