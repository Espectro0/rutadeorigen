package health

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

type Checker interface {
	Name() string
	Check(ctx context.Context) error
}

type checkerFunc struct {
	name string
	fn   func(ctx context.Context) error
}

func (c checkerFunc) Name() string                    { return c.name }
func (c checkerFunc) Check(ctx context.Context) error { return c.fn(ctx) }

func NewChecker(name string, fn func(ctx context.Context) error) Checker {
	return checkerFunc{name: name, fn: fn}
}

type Health struct {
	ready    atomic.Bool
	checkers []Checker
	timeout  time.Duration
	log      *slog.Logger
}

func New(log *slog.Logger, timeout time.Duration, checkers ...Checker) *Health {
	return &Health{log: log, timeout: timeout, checkers: checkers}
}

func (h *Health) SetReady(v bool) { h.ready.Store(v) }

func (h *Health) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /healthz", h.liveness)
	mux.HandleFunc("GET /readyz", h.readiness)
}

type response struct {
	Status string            `json:"status"`
	Checks map[string]string `json:"checks,omitempty"`
}

func (h *Health) liveness(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, response{Status: "ok"})
}

func (h *Health) readiness(w http.ResponseWriter, r *http.Request) {
	if !h.ready.Load() {
		writeJSON(w, http.StatusServiceUnavailable, response{Status: "not_ready"})
		return
	}

	checks, ok := h.runChecks(r.Context())

	if ok {
		writeJSON(w, 200, response{Status: "ok", Checks: checks})
	} else {
		writeJSON(w, 503, response{Status: "fail", Checks: checks})
	}
}

func (h *Health) runChecks(ctx context.Context) (map[string]string, bool) {
	ctx, cancel := context.WithTimeout(ctx, h.timeout)
	defer cancel()

	results := make([]error, len(h.checkers))
	var wg sync.WaitGroup
	for i, c := range h.checkers {
		wg.Go(func() { results[i] = c.Check(ctx) })
	}
	wg.Wait()

	checks := make(map[string]string, len(h.checkers))
	ok := true
	for i, c := range h.checkers {
		if results[i] == nil {
			checks[c.Name()] = "ok"
		} else {
			checks[c.Name()] = "fail"
			ok = false
			h.log.Warn("readiness check failed", "check", c.Name(), "err", results[i])
		}
	}

	return checks, ok
}

func writeJSON(w http.ResponseWriter, status int, body response) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
