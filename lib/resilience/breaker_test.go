package resilience_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/sony/gobreaker"

	"rutadeorigen/lib/resilience"
)

var errBoom = errors.New("boom")

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func fail(b *gobreaker.CircuitBreaker, err error) {
	_, _ = b.Execute(func() (any, error) { return nil, err })
}

func succeed(b *gobreaker.CircuitBreaker) {
	_, _ = b.Execute(func() (any, error) { return "ok", nil })
}

func TestNewBreakerStartsClosedWithItsName(t *testing.T) {
	b := resilience.NewBreaker("workos", discardLogger())

	if b.Name() != "workos" {
		t.Errorf("Name = %q, want %q", b.Name(), "workos")
	}
	if b.State() != gobreaker.StateClosed {
		t.Errorf("State = %v, want closed", b.State())
	}
}

func TestNewBreakerOpensAfterFiveConsecutiveFailures(t *testing.T) {
	b := resilience.NewBreaker("svc", discardLogger())

	for i := 1; i <= 4; i++ {
		fail(b, errBoom)
		if b.State() != gobreaker.StateClosed {
			t.Fatalf("State after %d failures = %v, want closed", i, b.State())
		}
	}

	fail(b, errBoom)

	if b.State() != gobreaker.StateOpen {
		t.Fatalf("State after 5 failures = %v, want open", b.State())
	}
}

func TestNewBreakerRejectsCallsWhileOpen(t *testing.T) {
	b := resilience.NewBreaker("svc", discardLogger())
	for i := 0; i < 5; i++ {
		fail(b, errBoom)
	}
	called := false

	_, err := b.Execute(func() (any, error) {
		called = true
		return nil, nil
	})

	if !errors.Is(err, gobreaker.ErrOpenState) {
		t.Errorf("error = %v, want %v", err, gobreaker.ErrOpenState)
	}
	if called {
		t.Error("the protected call must not run while the circuit is open")
	}
}

func TestNewBreakerSuccessResetsConsecutiveFailures(t *testing.T) {
	b := resilience.NewBreaker("svc", discardLogger())

	for i := 0; i < 4; i++ {
		fail(b, errBoom)
	}
	succeed(b)
	for i := 0; i < 4; i++ {
		fail(b, errBoom)
	}

	if b.State() != gobreaker.StateClosed {
		t.Fatalf("State = %v, want closed because failures were not consecutive", b.State())
	}

	fail(b, errBoom)

	if b.State() != gobreaker.StateOpen {
		t.Fatalf("State = %v, want open after 5 consecutive failures", b.State())
	}
}

func TestNewBreakerIgnoresCanceledCalls(t *testing.T) {
	b := resilience.NewBreaker("svc", discardLogger())

	for i := 0; i < 20; i++ {
		fail(b, context.Canceled)
	}

	if b.State() != gobreaker.StateClosed {
		t.Fatalf("State = %v, want closed because cancellations are not dependency failures", b.State())
	}
}

func TestNewBreakerCountsWrappedCancellationAsSuccess(t *testing.T) {
	b := resilience.NewBreaker("svc", discardLogger())

	for i := 0; i < 20; i++ {
		fail(b, errors.Join(errBoom, context.Canceled))
	}

	if b.State() != gobreaker.StateClosed {
		t.Fatalf("State = %v, want closed", b.State())
	}
}

func TestNewBreakerCountsTimeoutsAsFailures(t *testing.T) {
	b := resilience.NewBreaker("svc", discardLogger())

	for i := 0; i < 5; i++ {
		fail(b, context.DeadlineExceeded)
	}

	if b.State() != gobreaker.StateOpen {
		t.Fatalf("State = %v, want open", b.State())
	}
}

func TestNewBreakerLogsStateChanges(t *testing.T) {
	var logs bytes.Buffer
	b := resilience.NewBreaker("workos", slog.New(slog.NewTextHandler(&logs, nil)))

	for i := 0; i < 5; i++ {
		fail(b, errBoom)
	}

	out := logs.String()
	for _, want := range []string{"circuit breaker state changed", "workos", "closed", "open"} {
		if !strings.Contains(out, want) {
			t.Errorf("log %q must contain %q", out, want)
		}
	}
}

func TestNewBreakerDoesNotLogWhileNothingChanges(t *testing.T) {
	var logs bytes.Buffer
	b := resilience.NewBreaker("svc", slog.New(slog.NewTextHandler(&logs, nil)))

	for i := 0; i < 4; i++ {
		fail(b, errBoom)
	}
	succeed(b)

	if logs.Len() != 0 {
		t.Errorf("unexpected log output: %q", logs.String())
	}
}
