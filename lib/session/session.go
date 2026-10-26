package session

import "context"

type Session interface {
	Authenticate() (userID string, err error)
	Refresh(ctx context.Context) (sealed string, userID string, err error)
}

type Factory func(sealed string) Session
