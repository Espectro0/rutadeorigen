package controllers

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"

	"rutadeorigen/lib/session"
)

type LoginFlow interface {
	AuthorizationURL(state string) string
	Exchange(ctx context.Context, code string) (session.Result, error)
	LogoutURL(ctx context.Context, sealed, returnTo string) (string, error)
}

type StateGuard interface {
	Issue(w http.ResponseWriter) (string, error)
	Verify(w http.ResponseWriter, r *http.Request, received string) error
}

type SessionCookie interface {
	Read(r *http.Request) (string, bool)
	Set(w http.ResponseWriter, sealed string) error
	Clear(w http.ResponseWriter)
}

type AuthRedirects struct {
	Success string
	Failure string
	Logout  string
}

type AuthController struct {
	flow      LoginFlow
	state     StateGuard
	cookie    SessionCookie
	redirects AuthRedirects
	log       *slog.Logger
}

func NewAuthController(flow LoginFlow, state StateGuard, cookie SessionCookie, redirects AuthRedirects, log *slog.Logger) (*AuthController, error) {
	for name, raw := range map[string]string{
		"success": redirects.Success,
		"failure": redirects.Failure,
		"logout":  redirects.Logout,
	} {
		if err := validateRedirect(raw); err != nil {
			return nil, fmt.Errorf("auth redirect %s: %w", name, err)
		}
	}
	return &AuthController{flow: flow, state: state, cookie: cookie, redirects: redirects, log: log}, nil
}

func (c *AuthController) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /auth/login", c.login)
	mux.HandleFunc("GET /auth/callback", c.callback)
	mux.HandleFunc("GET /auth/logout", c.logout)
}

func (c *AuthController) login(w http.ResponseWriter, r *http.Request) {
	state, err := c.state.Issue(w)
	if err != nil {
		c.fail(w, r, err)
		return
	}
	redirect(w, r, c.flow.AuthorizationURL(state))
}

func (c *AuthController) callback(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	if err := c.state.Verify(w, r, query.Get("state")); err != nil {
		c.fail(w, r, err)
		return
	}
	if query.Get("error") != "" {
		c.log.WarnContext(r.Context(), "authorization denied by provider", "error", query.Get("error"))
		c.redirectFailure(w, r, "denied")
		return
	}

	result, err := c.flow.Exchange(r.Context(), query.Get("code"))
	if err != nil {
		c.fail(w, r, err)
		return
	}
	if err := c.cookie.Set(w, result.Sealed); err != nil {
		c.fail(w, r, err)
		return
	}

	redirect(w, r, c.redirects.Success)
}

func (c *AuthController) logout(w http.ResponseWriter, r *http.Request) {
	sealed, found := c.cookie.Read(r)
	c.cookie.Clear(w)

	target := c.redirects.Logout
	if found {
		providerURL, err := c.flow.LogoutURL(r.Context(), sealed, c.redirects.Logout)
		if err != nil {
			c.log.WarnContext(r.Context(), "provider logout unavailable", "err", err)
		} else {
			target = providerURL
		}
	}

	redirect(w, r, target)
}

func (c *AuthController) fail(w http.ResponseWriter, r *http.Request, err error) {
	reason := failureReason(err)
	c.log.WarnContext(r.Context(), "authentication failed", "reason", reason, "err", err)
	c.redirectFailure(w, r, reason)
}

func (c *AuthController) redirectFailure(w http.ResponseWriter, r *http.Request, reason string) {
	target, err := url.Parse(c.redirects.Failure)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	query := target.Query()
	query.Set("error", reason)
	target.RawQuery = query.Encode()
	redirect(w, r, target.String())
}

func redirect(w http.ResponseWriter, r *http.Request, target string) {
	w.Header().Set("Cache-Control", "no-store")
	http.Redirect(w, r, target, http.StatusFound) //nolint:gosec // target is a configured url or the provider logout url, never request input
}

func failureReason(err error) string {
	switch {
	case errors.Is(err, session.ErrInvalidState):
		return "invalid_state"
	case errors.Is(err, session.ErrUnauthenticated):
		return "unauthenticated"
	case errors.Is(err, session.ErrAuthUnavailable):
		return "unavailable"
	default:
		return "internal"
	}
}

func validateRedirect(raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("parse redirect url: %w", err)
	}
	if (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return fmt.Errorf("%q must be an absolute http(s) url", raw)
	}
	return nil
}
