package session

import (
	"context"
	"errors"
	"fmt"

	"github.com/workos/workos-go/v10"

	"rutadeorigen/lib/requestcontext"
)

type workosSession struct {
	inner *workos.Session
}

func NewWorkOSFactory(client *workos.Client, cookiePassword string) Factory {
	return func(sealed string) Session {
		return &workosSession{inner: workos.NewSession(client, sealed, cookiePassword)}
	}
}

func (s *workosSession) Authenticate() (requestcontext.Identity, error) {
	result, err := s.inner.Authenticate()
	if errors.Is(err, workos.ErrSigningKeyUnavailable) {
		return requestcontext.Identity{}, fmt.Errorf("%w: %w", ErrAuthUnavailable, err)
	}
	if err != nil || result == nil || !result.Authenticated {
		return requestcontext.Identity{}, ErrUnauthenticated
	}
	if result.User == nil || result.User.ID == "" {
		return requestcontext.Identity{}, ErrUnauthenticated
	}
	return requestcontext.Identity{UserID: result.User.ID, Email: result.User.Email}, nil
}

func (s *workosSession) Refresh(ctx context.Context) (string, requestcontext.Identity, error) {
	result, err := s.inner.Refresh(ctx)
	if err != nil {
		if result != nil && result.Reason == reasonRefreshTokenRevoked {
			return "", requestcontext.Identity{}, ErrUnauthenticated
		}
		return "", requestcontext.Identity{}, fmt.Errorf("%w: %w", ErrAuthUnavailable, err)
	}
	if result == nil || !result.Authenticated || result.Session == nil {
		return "", requestcontext.Identity{}, ErrUnauthenticated
	}
	if result.Session.User == nil || result.Session.User.ID == "" {
		return "", requestcontext.Identity{}, ErrUnauthenticated
	}

	identity := requestcontext.Identity{UserID: result.Session.User.ID, Email: result.Session.User.Email}
	return result.SealedSession, identity, nil
}
