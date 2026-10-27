package resilience

import (
	"errors"
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
		resp, err := t.base.RoundTrip(req)
		if err != nil {
			return nil, err
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
		return nil, err
	}
	return result.(*http.Response), nil
}
