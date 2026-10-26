package session

import (
	"context"
	"errors"
	"fmt"

	"github.com/workos/workos-go/v10"
)

type Result struct {
	Sealed string
	UserID string
	Email  string
}

type Login struct {
	client         *workos.Client
	redirectURI    string
	cookiePassword string
}

func NewLogin(client *workos.Client, redirectURI, cookiePassword string) *Login {
	return &Login{
		client:         client,
		redirectURI:    redirectURI,
		cookiePassword: cookiePassword,
	}
}

func (l *Login) AuthorizationURL(state string) string {
	provider := workos.UserManagementAuthenticationProvider("authkit")
	return l.client.UserManagement().GetAuthorizationURL(&workos.UserManagementGetAuthorizationURLParams{
		Provider:    &provider,
		State:       &state,
		RedirectURI: l.redirectURI,
	})
}

func (l *Login) Exchange(ctx context.Context, code string) (Result, error) {
	if code == "" {
		return Result{}, ErrUnauthenticated
	}

	resp, err := l.client.UserManagement().AuthenticateWithCode(ctx, &workos.UserManagementAuthenticateWithCodeParams{
		Code: code,
	})
	if err != nil {
		return Result{}, classifyExchangeError(err)
	}
	if resp == nil || resp.User == nil || resp.User.ID == "" || resp.AccessToken == "" {
		return Result{}, ErrUnauthenticated
	}

	sealed, err := workos.SealSessionFromAuthResponse(
		resp.AccessToken,
		resp.RefreshToken,
		resp.User,
		resp.Impersonator,
		l.cookiePassword,
	)
	if err != nil {
		return Result{}, fmt.Errorf("seal session: %w", err)
	}

	return Result{
		Sealed: sealed,
		UserID: resp.User.ID,
		Email:  resp.User.Email,
	}, nil
}

func (l *Login) LogoutURL(ctx context.Context, sealed, returnTo string) (string, error) {
	if sealed == "" {
		return "", ErrUnauthenticated
	}

	url, err := workos.NewSession(l.client, sealed, l.cookiePassword).GetLogoutURL(ctx, returnTo)
	if errors.Is(err, workos.ErrSigningKeyUnavailable) {
		return "", fmt.Errorf("%w: %w", ErrAuthUnavailable, err)
	}
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrUnauthenticated, err)
	}

	return url, nil
}

func classifyExchangeError(err error) error {
	var apiErr *workos.APIError
	if errors.As(err, &apiErr) && apiErr.StatusCode >= 400 && apiErr.StatusCode < 500 {
		return fmt.Errorf("%w: %w", ErrUnauthenticated, err)
	}
	return fmt.Errorf("%w: %w", ErrAuthUnavailable, err)
}
