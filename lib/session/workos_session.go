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

func NewWorkOSFactory(client *workos.Client, cookiePassword string, opts ...workos.SessionOption) Factory {
	return func(sealed string) Session {
		return &workosSession{inner: workos.NewSession(client, sealed, cookiePassword, opts...)}
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
	return identityFrom(result.User), nil
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
	return result.SealedSession, identityFrom(result.Session.User), nil
}

func identityFrom(user *workos.User) requestcontext.Identity {
	return requestcontext.Identity{
		UserID:    user.ID,
		Email:     user.Email,
		FirstName: user.FirstName,
		LastName:  user.LastName,
	}
}
