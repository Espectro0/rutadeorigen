package session

import (
	"context"
	"rutadeorigen/lib/requestcontext"
)

type Session interface {
	Authenticate() (requestcontext.Identity, error)
	Refresh(ctx context.Context) (sealed string, identity requestcontext.Identity, err error)
}

type Factory func(sealed string) Session
