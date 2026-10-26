package session

import "errors"

var (
	ErrCookieTooLarge  = errors.New("session cookie exceeds size limit")
	ErrUnauthenticated = errors.New("unauthenticated")
	ErrAuthUnavailable = errors.New("authentication unavailable")
	ErrInvalidState    = errors.New("invalid oauth state")
)

const (
	reasonRefreshTokenRevoked = "refresh_token_revoked"
)
