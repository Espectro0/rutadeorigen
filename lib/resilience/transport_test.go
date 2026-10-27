package resilience_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sony/gobreaker"

	"rutadeorigen/lib/resilience"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

type backend struct {
	*httptest.Server
	status atomic.Int32
	hits   atomic.Int32
}

func newBackend(t *testing.T, status int) *backend {
	t.Helper()

	b := &backend{}
	b.status.Store(int32(status))
	b.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		b.hits.Add(1)
		w.WriteHeader(int(b.status.Load()))
		_, _ = w.Write([]byte("payload"))
	}))
	t.Cleanup(b.Close)
	return b
}

func get(t *testing.T, client *http.Client, target string) (*http.Response, error) {
	t.Helper()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, target, nil)
	if err != nil {
		t.Fatal(err)
	}
	return client.Do(req)
}

func getErr(t *testing.T, client *http.Client, target string) error {
	t.Helper()

	resp, err := get(t, client, target)
	if err != nil {
		return err
	}
	drain(t, resp)
	return nil
}

func drain(t *testing.T, resp *http.Response) string {
	t.Helper()

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}

func clientWith(breaker *gobreaker.CircuitBreaker, base http.RoundTripper) *http.Client {
	return &http.Client{Transport: resilience.NewTransport(breaker, base)}
}

func quickBreaker(timeout time.Duration) *gobreaker.CircuitBreaker {
	return gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        "quick",
		MaxRequests: 1,
		Timeout:     timeout,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= 2
		},
	})
}

func TestTransportPassesSuccessfulResponsesThrough(t *testing.T) {
	srv := newBackend(t, http.StatusOK)
	client := clientWith(resilience.NewBreaker("svc", discardLogger()), nil)

	resp, err := get(t, client, srv.URL)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if body := drain(t, resp); body != "payload" {
		t.Errorf("body = %q, want %q", body, "payload")
	}
}

func TestTransportReturnsServerErrorsToTheCaller(t *testing.T) {
	for _, status := range []int{http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusTooManyRequests} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			srv := newBackend(t, status)
			client := clientWith(resilience.NewBreaker("svc", discardLogger()), nil)

			resp, err := get(t, client, srv.URL)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if resp.StatusCode != status {
				t.Errorf("status = %d, want %d", resp.StatusCode, status)
			}
			if body := drain(t, resp); body != "payload" {
				t.Errorf("body = %q, want the original body", body)
			}
		})
	}
}

func TestTransportOpensAfterConsecutiveServerErrors(t *testing.T) {
	for _, status := range []int{http.StatusInternalServerError, http.StatusServiceUnavailable, http.StatusTooManyRequests} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			srv := newBackend(t, status)
			breaker := resilience.NewBreaker("svc", discardLogger())
			client := clientWith(breaker, nil)

			for i := 0; i < 5; i++ {
				resp, err := get(t, client, srv.URL)
				if err != nil {
					t.Fatalf("request %d: unexpected error: %v", i+1, err)
				}
				drain(t, resp)
			}

			if breaker.State() != gobreaker.StateOpen {
				t.Fatalf("State = %v, want open", breaker.State())
			}

			err := getErr(t, client, srv.URL)

			if !errors.Is(err, gobreaker.ErrOpenState) {
				t.Errorf("error = %v, want %v", err, gobreaker.ErrOpenState)
			}
			if hits := srv.hits.Load(); hits != 5 {
				t.Errorf("backend hits = %d, want 5 because the open circuit must not reach it", hits)
			}
		})
	}
}

func TestTransportDoesNotCountClientErrors(t *testing.T) {
	srv := newBackend(t, http.StatusBadRequest)
	breaker := resilience.NewBreaker("svc", discardLogger())
	client := clientWith(breaker, nil)

	for i := 0; i < 20; i++ {
		resp, err := get(t, client, srv.URL)
		if err != nil {
			t.Fatalf("request %d: unexpected error: %v", i+1, err)
		}
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("request %d: status = %d, want %d", i+1, resp.StatusCode, http.StatusBadRequest)
		}
		drain(t, resp)
	}

	if breaker.State() != gobreaker.StateClosed {
		t.Errorf("State = %v, want closed because 4xx are not dependency failures", breaker.State())
	}
	if hits := srv.hits.Load(); hits != 20 {
		t.Errorf("backend hits = %d, want 20", hits)
	}
}

func TestTransportOpensAfterTransportErrors(t *testing.T) {
	var calls atomic.Int32
	base := roundTripFunc(func(*http.Request) (*http.Response, error) {
		calls.Add(1)
		return nil, errBoom
	})
	breaker := resilience.NewBreaker("svc", discardLogger())
	client := clientWith(breaker, base)

	for i := 0; i < 5; i++ {
		err := getErr(t, client, "http://dependency.invalid/")
		if !errors.Is(err, errBoom) {
			t.Fatalf("request %d: error = %v, want %v", i+1, err, errBoom)
		}
	}
	err := getErr(t, client, "http://dependency.invalid/")

	if !errors.Is(err, gobreaker.ErrOpenState) {
		t.Errorf("error = %v, want %v", err, gobreaker.ErrOpenState)
	}
	if got := calls.Load(); got != 5 {
		t.Errorf("base calls = %d, want 5", got)
	}
}

func TestTransportIgnoresCanceledRequests(t *testing.T) {
	base := roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, context.Canceled
	})
	breaker := resilience.NewBreaker("svc", discardLogger())
	client := clientWith(breaker, base)

	for i := 0; i < 20; i++ {
		_ = getErr(t, client, "http://dependency.invalid/")
	}

	if breaker.State() != gobreaker.StateClosed {
		t.Errorf("State = %v, want closed because the caller gave up, not the dependency", breaker.State())
	}
}

func TestTransportCountsTimeouts(t *testing.T) {
	base := roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, context.DeadlineExceeded
	})
	breaker := resilience.NewBreaker("svc", discardLogger())
	client := clientWith(breaker, base)

	for i := 0; i < 5; i++ {
		_ = getErr(t, client, "http://dependency.invalid/")
	}

	if breaker.State() != gobreaker.StateOpen {
		t.Errorf("State = %v, want open", breaker.State())
	}
}

func TestTransportSuccessResetsTheFailureCount(t *testing.T) {
	srv := newBackend(t, http.StatusInternalServerError)
	breaker := resilience.NewBreaker("svc", discardLogger())
	client := clientWith(breaker, nil)

	run := func(status, times int) {
		srv.status.Store(int32(status))
		for i := 0; i < times; i++ {
			resp, err := get(t, client, srv.URL)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			drain(t, resp)
		}
	}

	run(http.StatusInternalServerError, 4)
	run(http.StatusOK, 1)
	run(http.StatusInternalServerError, 4)

	if breaker.State() != gobreaker.StateClosed {
		t.Errorf("State = %v, want closed because failures were not consecutive", breaker.State())
	}
}

func TestTransportRecoversThroughHalfOpen(t *testing.T) {
	srv := newBackend(t, http.StatusInternalServerError)
	breaker := quickBreaker(50 * time.Millisecond)
	client := clientWith(breaker, nil)

	for i := 0; i < 2; i++ {
		resp, err := get(t, client, srv.URL)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		drain(t, resp)
	}
	if breaker.State() != gobreaker.StateOpen {
		t.Fatalf("State = %v, want open", breaker.State())
	}

	srv.status.Store(http.StatusOK)
	time.Sleep(80 * time.Millisecond)

	resp, err := get(t, client, srv.URL)
	if err != nil {
		t.Fatalf("the probe request must be allowed, got %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	drain(t, resp)

	if breaker.State() != gobreaker.StateClosed {
		t.Errorf("State = %v, want closed after a successful probe", breaker.State())
	}
}

func TestTransportReopensWhenTheProbeFails(t *testing.T) {
	srv := newBackend(t, http.StatusInternalServerError)
	breaker := quickBreaker(50 * time.Millisecond)
	client := clientWith(breaker, nil)

	for i := 0; i < 2; i++ {
		resp, err := get(t, client, srv.URL)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		drain(t, resp)
	}
	time.Sleep(80 * time.Millisecond)

	resp, err := get(t, client, srv.URL)
	if err != nil {
		t.Fatalf("the probe request must be allowed, got %v", err)
	}
	drain(t, resp)

	if breaker.State() != gobreaker.StateOpen {
		t.Fatalf("State = %v, want open after a failed probe", breaker.State())
	}
	hits := srv.hits.Load()

	err = getErr(t, client, srv.URL)

	if !errors.Is(err, gobreaker.ErrOpenState) {
		t.Errorf("error = %v, want %v", err, gobreaker.ErrOpenState)
	}
	if srv.hits.Load() != hits {
		t.Error("a reopened circuit must not reach the backend")
	}
}

func TestTransportUsesTheDefaultTransportWhenBaseIsNil(t *testing.T) {
	srv := newBackend(t, http.StatusOK)
	client := &http.Client{Transport: resilience.NewTransport(quickBreaker(time.Second), nil)}

	resp, err := get(t, client, srv.URL)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	drain(t, resp)
	if srv.hits.Load() != 1 {
		t.Errorf("backend hits = %d, want 1", srv.hits.Load())
	}
}

func TestTransportSharesOneBreakerAcrossClients(t *testing.T) {
	srv := newBackend(t, http.StatusInternalServerError)
	breaker := resilience.NewBreaker("svc", discardLogger())
	first := clientWith(breaker, nil)
	second := clientWith(breaker, nil)

	for i := 0; i < 5; i++ {
		resp, err := get(t, first, srv.URL)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		drain(t, resp)
	}

	if err := getErr(t, second, srv.URL); !errors.Is(err, gobreaker.ErrOpenState) {
		t.Errorf("error = %v, want %v", err, gobreaker.ErrOpenState)
	}
}
