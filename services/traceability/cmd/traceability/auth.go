package main

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/workos/workos-go/v10"

	"rutadeorigen/lib/config"
	libmiddleware "rutadeorigen/lib/httpserver/middleware"
	"rutadeorigen/lib/resilience"
	"rutadeorigen/lib/session"
	"rutadeorigen/traceability/internal/infrastructure/input/controllers"
)

type authMode string

const (
	authModeOIDC      authMode = "oidc"
	authModeDevHeader authMode = "dev-header"

	minCookiePasswordLength = 32
)

type authn struct {
	mode          authMode
	authenticator libmiddleware.Authenticator
	routes        *controllers.AuthController
}

func loadAuth(log *slog.Logger, storage string) (authn, error) {
	switch mode := authMode(config.String("AUTH_MODE", string(authModeOIDC))); mode {
	case authModeDevHeader:
		if storage != "memory" {
			return authn{}, errors.New("AUTH_MODE=dev-header requires STORAGE=memory")
		}
		log.Warn("AUTH_MODE=dev-header, trusting the X-Actor-ID header (development only)")
		return authn{mode: authModeDevHeader}, nil
	case authModeOIDC:
		return loadOIDC(log)
	default:
		return authn{}, fmt.Errorf("unknown AUTH_MODE %q", mode)
	}
}

func loadOIDC(log *slog.Logger) (authn, error) {
	var errs []error
	required := func(key string) string {
		v, err := config.Required(key)
		if err != nil {
			errs = append(errs, err)
		}
		return v
	}

	apiKey := required("WORKOS_API_KEY")
	clientID := required("WORKOS_CLIENT_ID")
	redirectURI := required("WORKOS_REDIRECT_URI")
	cookiePassword := required("WORKOS_COOKIE_PASSWORD")
	redirects := controllers.AuthRedirects{
		Success: required("AUTH_SUCCESS_URL"),
		Failure: required("AUTH_FAILURE_URL"),
		Logout:  required("AUTH_LOGOUT_URL"),
	}
	domain := config.String("COOKIE_DOMAIN", "")
	secure, err := config.Bool("COOKIE_SECURE", true)
	if err != nil {
		errs = append(errs, err)
	}
	if cookiePassword != "" && len(cookiePassword) < minCookiePasswordLength {
		errs = append(errs, fmt.Errorf("WORKOS_COOKIE_PASSWORD must have at least %d characters", minCookiePasswordLength))
	}
	if joined := errors.Join(errs...); joined != nil {
		return authn{}, joined
	}

	breaker := resilience.NewBreaker("workos", log)
	httpClient := &http.Client{
		Timeout:   5 * time.Second,
		Transport: resilience.NewTransport(breaker, http.DefaultTransport),
	}
	client := workos.NewClient(apiKey,
		workos.WithClientID(clientID),
		workos.WithHTTPClient(httpClient),
		workos.WithMaxRetries(1),
	)
	cookie := session.NewCookie(session.CookieConfig{Domain: domain, Secure: secure})
	state := session.NewState(session.StateConfig{Domain: domain, Secure: secure})
	login := session.NewLogin(client, redirectURI, cookiePassword)
	authenticator := session.NewAuthenticator(session.NewWorkOSFactory(client, cookiePassword), cookie)

	routes, err := controllers.NewAuthController(login, state, cookie, redirects, log)
	if err != nil {
		return authn{}, err
	}
	return authn{mode: authModeOIDC, authenticator: authenticator, routes: routes}, nil
}

func (a authn) middlewares(errs *controllers.ErrorResponder, resolver controllers.ActorResolver) (requireActor, authenticated controllers.Middleware) {
	if a.mode == authModeDevHeader {
		return func(next http.Handler) http.Handler {
			return controllers.RequireActorHeader(errs, next)
		}, nil
	}

	authenticated = libmiddleware.Authenticate(a.authenticator, controllers.RejectSession(errs))
	requireActor = func(next http.Handler) http.Handler {
		return authenticated(controllers.RequireActor(resolver, errs, next))
	}
	return requireActor, authenticated
}
