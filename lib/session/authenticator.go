package session

import (
	"errors"
	"net/http"

	"rutadeorigen/lib/requestcontext"
)

type Option func(*Authenticator)

func WithoutRefresh() Option {
	return func(a *Authenticator) { a.refresh = false }
}

type Authenticator struct {
	factory Factory
	cookie  Cookie
	refresh bool
}

func NewAuthenticator(factory Factory, cookie Cookie, opts ...Option) *Authenticator {
	a := &Authenticator{factory: factory, cookie: cookie, refresh: true}
	for _, opt := range opts {
		opt(a)
	}
	return a
}

func (a *Authenticator) Authenticate(w http.ResponseWriter, r *http.Request) (requestcontext.Identity, error) {
	sealed, ok := a.cookie.Read(r)
	if !ok {
		return requestcontext.Identity{}, ErrUnauthenticated
	}

	current := a.factory(sealed)

	identity, err := current.Authenticate()
	if err == nil {
		return identity, nil
	}
	if errors.Is(err, ErrAuthUnavailable) {
		return requestcontext.Identity{}, err
	}

	if !a.refresh {
		return requestcontext.Identity{}, ErrUnauthenticated
	}

	refreshed, identity, err := current.Refresh(r.Context())
	if err != nil {
		if errors.Is(err, ErrUnauthenticated) {
			a.cookie.Clear(w)
		}
		return requestcontext.Identity{}, err
	}

	if err := a.cookie.Set(w, refreshed); err != nil {
		return requestcontext.Identity{}, err
	}

	return identity, nil
}
