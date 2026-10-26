package session

import (
	"errors"
	"net/http"
)

type Authenticator struct {
	factory Factory
	cookie  Cookie
}

func NewAuthenticator(factory Factory, cookie Cookie) *Authenticator {
	return &Authenticator{
		factory: factory,
		cookie:  cookie,
	}
}

func (a *Authenticator) Authenticate(w http.ResponseWriter, r *http.Request) (string, error) {
	sealed, ok := a.cookie.Read(r)
	if !ok {
		return "", ErrUnauthenticated
	}

	current := a.factory(sealed)

	userID, err := current.Authenticate()
	if err == nil {
		return userID, nil
	}
	if errors.Is(err, ErrAuthUnavailable) {
		return "", err
	}

	refreshed, userID, err := current.Refresh(r.Context())
	if err != nil {
		if errors.Is(err, ErrUnauthenticated) {
			a.cookie.Clear(w)
		}
		return "", err
	}

	if err := a.cookie.Set(w, refreshed); err != nil {
		return "", err
	}

	return userID, nil
}
