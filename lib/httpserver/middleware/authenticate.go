package middleware

import (
	"net/http"

	"rutadeorigen/lib/requestcontext"
)

type Authenticator interface {
	Authenticate(w http.ResponseWriter, r *http.Request) (string, error)
}

type RejectFunc func(w http.ResponseWriter, r *http.Request, err error)

func Authenticate(auth Authenticator, reject RejectFunc) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, err := auth.Authenticate(w, r)
			if err != nil {
				reject(w, r, err)
				return
			}
			next.ServeHTTP(w, r.WithContext(requestcontext.WithUserID(r.Context(), userID)))
		})
	}
}
