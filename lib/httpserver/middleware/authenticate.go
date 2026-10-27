package middleware

import (
	"net/http"

	"rutadeorigen/lib/requestcontext"
)

type Authenticator interface {
	Authenticate(w http.ResponseWriter, r *http.Request) (requestcontext.Identity, error)
}

type RejectFunc func(w http.ResponseWriter, r *http.Request, err error)

func Authenticate(auth Authenticator, reject RejectFunc) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			identity, err := auth.Authenticate(w, r)
			if err != nil {
				reject(w, r, err)
				return
			}

			ctx := requestcontext.WithUserID(r.Context(), identity.UserID)
			ctx = requestcontext.WithEmail(ctx, identity.Email)
			if identity.FirstName != nil {
				ctx = requestcontext.WithFirstName(ctx, *identity.FirstName)
			}
			if identity.LastName != nil {
				ctx = requestcontext.WithLastName(ctx, *identity.LastName)
			}
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
