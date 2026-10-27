package session

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sony/gobreaker"

	"rutadeorigen/lib/resilience"
)

func newProtectedFactory(ws *workosServer) (Factory, *gobreaker.CircuitBreaker) {
	breaker := resilience.NewBreaker("workos", slog.New(slog.NewTextHandler(io.Discard, nil)))
	return NewWorkOSFactory(ws.protectedClient(breaker), testCookiePassword), breaker
}

func TestSigningKeyRequestsGoThroughTheBreakerAndTheSDKLimitsRetries(t *testing.T) {
	ws := newWorkOSServer(t)
	ws.setJWKSStatus(http.StatusInternalServerError)
	factory, breaker := newProtectedFactory(ws)

	for i := 0; i < 8; i++ {
		_, err := factory(validSession(t, "user_1")).Authenticate()
		if !errors.Is(err, ErrAuthUnavailable) {
			t.Fatalf("attempt %d: error = %v, want %v", i+1, err, ErrAuthUnavailable)
		}
	}

	if hits := ws.jwksRequests(); hits != 1 {
		t.Errorf("signing key requests = %d, want 1 because the SDK holds a retry cooldown after a failed fetch", hits)
	}
	if failures := breaker.Counts().TotalFailures; failures != 1 {
		t.Errorf("breaker failures = %d, want 1 because the signing key request must go through the breaker", failures)
	}
	if breaker.State() != gobreaker.StateClosed {
		t.Errorf("State = %v, want closed", breaker.State())
	}
}

func TestBreakerStopsRefreshingWhenWorkOSKeepsFailing(t *testing.T) {
	ws := newWorkOSServer(t)
	ws.setAuthResponse(http.StatusInternalServerError, map[string]string{"message": "boom"})
	factory, breaker := newProtectedFactory(ws)

	for i := 0; i < 8; i++ {
		_, _, err := factory(expiredSession(t, "user_1")).Refresh(context.Background())
		if !errors.Is(err, ErrAuthUnavailable) {
			t.Fatalf("attempt %d: error = %v, want %v", i+1, err, ErrAuthUnavailable)
		}
	}

	if breaker.State() != gobreaker.StateOpen {
		t.Fatalf("State = %v, want open", breaker.State())
	}
	if got := len(ws.requests()); got != 5 {
		t.Errorf("refresh requests = %d, want 5", got)
	}
}

func TestBreakerDoesNotOpenForRevokedRefreshTokens(t *testing.T) {
	ws := newWorkOSServer(t)
	ws.setAuthResponse(http.StatusBadRequest, map[string]string{"error": "invalid_grant", "error_description": "revoked"})
	factory, breaker := newProtectedFactory(ws)

	for i := 0; i < 12; i++ {
		_, _, err := factory(expiredSession(t, "user_1")).Refresh(context.Background())
		if !errors.Is(err, ErrUnauthenticated) {
			t.Fatalf("attempt %d: error = %v, want %v", i+1, err, ErrUnauthenticated)
		}
	}

	if breaker.State() != gobreaker.StateClosed {
		t.Errorf("State = %v, want closed because a revoked token is not a WorkOS failure", breaker.State())
	}
	if got := len(ws.requests()); got != 12 {
		t.Errorf("refresh requests = %d, want 12", got)
	}
}

func TestBreakerLoginExchangeReportsUnavailableWhenOpen(t *testing.T) {
	ws := newWorkOSServer(t)
	ws.setAuthResponse(http.StatusInternalServerError, map[string]string{"message": "boom"})
	breaker := resilience.NewBreaker("workos", slog.New(slog.NewTextHandler(io.Discard, nil)))
	login := NewLogin(ws.protectedClient(breaker), "http://localhost:8080/auth/callback", testCookiePassword)

	for i := 0; i < 8; i++ {
		_, err := login.Exchange(context.Background(), "code-123")
		if !errors.Is(err, ErrAuthUnavailable) {
			t.Fatalf("attempt %d: error = %v, want %v", i+1, err, ErrAuthUnavailable)
		}
	}

	if got := len(ws.requests()); got != 5 {
		t.Errorf("exchange requests = %d, want 5", got)
	}
}

func TestBreakerAuthenticatorDoesNotRefreshWhenTheCircuitIsOpen(t *testing.T) {
	ws := newWorkOSServer(t)
	ws.setJWKSStatus(http.StatusInternalServerError)
	factory, _ := newProtectedFactory(ws)
	authenticator := NewAuthenticator(factory, NewCookie(CookieConfig{}))

	for i := 0; i < 8; i++ {
		_, err := authenticator.Authenticate(httptest.NewRecorder(), authRequest(expiredSession(t, "user_1")))
		if !errors.Is(err, ErrAuthUnavailable) {
			t.Fatalf("attempt %d: error = %v, want %v", i+1, err, ErrAuthUnavailable)
		}
	}

	if got := len(ws.requests()); got != 0 {
		t.Errorf("refresh requests = %d, want 0 so refresh tokens are never burned during an outage", got)
	}
}

func TestBreakerRecoversWhenWorkOSComesBack(t *testing.T) {
	ws := newWorkOSServer(t)
	breaker := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        "workos",
		MaxRequests: 1,
		Timeout:     50 * time.Millisecond,
		ReadyToTrip: func(counts gobreaker.Counts) bool { return counts.ConsecutiveFailures >= 2 },
	})
	factory := NewWorkOSFactory(ws.protectedClient(breaker), testCookiePassword)

	ws.setAuthResponse(http.StatusInternalServerError, map[string]string{"message": "boom"})
	for i := 0; i < 2; i++ {
		_, _, _ = factory(expiredSession(t, "user_1")).Refresh(context.Background())
	}
	if breaker.State() != gobreaker.StateOpen {
		t.Fatalf("State = %v, want open", breaker.State())
	}

	key, _ := keys(t)
	ws.setAuthResponse(http.StatusOK, authResponse(signToken(t, key, "session_1", 5*time.Minute), "refresh_2", "user_1"))
	time.Sleep(80 * time.Millisecond)

	_, identity, err := factory(expiredSession(t, "user_1")).Refresh(context.Background())

	if err != nil || identity.UserID != "user_1" {
		t.Fatalf("Refresh() = (%+v, %v), want user_1 once WorkOS is back", identity, err)
	}
	if breaker.State() != gobreaker.StateClosed {
		t.Errorf("State = %v, want closed", breaker.State())
	}
}
