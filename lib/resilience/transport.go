package resilience

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/sony/gobreaker"
)

var errServerStatus = errors.New("server error status")

type transport struct {
	breaker *gobreaker.CircuitBreaker
	base    http.RoundTripper
}

func NewTransport(breaker *gobreaker.CircuitBreaker, base http.RoundTripper) http.RoundTripper {
	if base == nil {
		base = http.DefaultTransport
	}
	return &transport{breaker: breaker, base: base}
}

func (t *transport) RoundTrip(req *http.Request) (*http.Response, error) {
	result, err := t.breaker.Execute(func() (any, error) {
		resp, err := t.base.RoundTrip(req) //nolint:bodyclose // the caller owns the response body
		if err != nil {
			return nil, fmt.Errorf("round trip: %w", err)
		}
		if resp.StatusCode >= http.StatusInternalServerError || resp.StatusCode == http.StatusTooManyRequests {
			return resp, errServerStatus
		}
		return resp, nil
	})
	if errors.Is(err, errServerStatus) {
		return result.(*http.Response), nil
	}
	if err != nil {
		return nil, fmt.Errorf("protected call: %w", err)
	}
	return result.(*http.Response), nil
}
