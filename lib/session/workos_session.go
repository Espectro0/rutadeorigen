package session

import (
	"context"
	"errors"
	"fmt"

	"github.com/workos/workos-go/v10"
)

type workosSession struct {
	inner *workos.Session
}

func NewWorkOSFactory(client *workos.Client, cookiePassword string) Factory {
	return func(sealed string) Session {
		return &workosSession{inner: workos.NewSession(client, sealed, cookiePassword)}
	}
}

func (s *workosSession) Authenticate() (string, error) {
	result, err := s.inner.Authenticate()
	if errors.Is(err, workos.ErrSigningKeyUnavailable) {
		return "", fmt.Errorf("%w: %w", ErrAuthUnavailable, err)
	}
	if err != nil || result == nil || !result.Authenticated {
		return "", ErrUnauthenticated
	}
	if result.User == nil || result.User.ID == "" {
		return "", ErrUnauthenticated
	}
	return result.User.ID, nil
}

func (s *workosSession) Refresh(ctx context.Context) (string, string, error) {
	result, err := s.inner.Refresh(ctx)
	if err != nil {
		if result != nil && result.Reason == reasonRefreshTokenRevoked {
			return "", "", ErrUnauthenticated
		}
		return "", "", fmt.Errorf("%w: %w", ErrAuthUnavailable, err)
	}
	if result == nil || !result.Authenticated || result.Session == nil {
		return "", "", ErrUnauthenticated
	}
	if result.Session.User == nil || result.Session.User.ID == "" {
		return "", "", ErrUnauthenticated
	}
	return result.SealedSession, result.Session.User.ID, nil
}
