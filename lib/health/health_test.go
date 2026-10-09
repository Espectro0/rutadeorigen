package health_test

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"rutadeorigen/lib/health"
)

type body struct {
	Status string            `json:"status"`
	Checks map[string]string `json:"checks"`
}

func newHealth(timeout time.Duration, checkers ...health.Checker) *health.Health {
	return health.New(slog.New(slog.DiscardHandler), timeout, checkers...)
}

func newMux(h *health.Health) *http.ServeMux {
	mux := http.NewServeMux()
	h.Register(mux)
	return mux
}

func get(t *testing.T, mux *http.ServeMux, path string) (int, body) {
	t.Helper()
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))

	var b body
	if err := json.Unmarshal(rec.Body.Bytes(), &b); err != nil {
		t.Fatalf("%s: response is not valid JSON: %v\n%s", path, err, rec.Body.String())
	}
	return rec.Code, b
}

var (
	okChecker = health.NewChecker("cache", func(context.Context) error {
		return nil
	})
	failChecker = health.NewChecker("db", func(context.Context) error {
		return errors.New("connection refused")
	})
)

func TestLiveness_AlwaysOK(t *testing.T) {
	h := newHealth(time.Second, failChecker)

	code, b := get(t, newMux(h), "/healthz")
	if code != http.StatusOK {
		t.Errorf("status = %d, expected 200", code)
	}
	if b.Status != "ok" {
		t.Errorf("body status = %q, expected ok", b.Status)
	}
}

func TestReadiness_NotReadyByDefault(t *testing.T) {
	h := newHealth(time.Second)

	code, b := get(t, newMux(h), "/readyz")
	if code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, expected 503", code)
	}
	if b.Status != "not_ready" {
		t.Errorf("body status = %q, expected not_ready", b.Status)
	}
}

func TestReadiness_ReadyWithoutCheckers(t *testing.T) {
	h := newHealth(time.Second)
	h.SetReady(true)

	code, b := get(t, newMux(h), "/readyz")
	if code != http.StatusOK {
		t.Errorf("status = %d, expected 200", code)
	}
	if b.Status != "ok" {
		t.Errorf("body status = %q, expected ok", b.Status)
	}
}

func TestReadiness_AllCheckersPass(t *testing.T) {
	h := newHealth(time.Second, okChecker)
	h.SetReady(true)

	code, b := get(t, newMux(h), "/readyz")
	if code != http.StatusOK {
		t.Errorf("status = %d, expected 200", code)
	}
	if b.Checks["cache"] != "ok" {
		t.Errorf("checks[cache] = %q, expected ok", b.Checks["cache"])
	}
}

func TestReadiness_FailingChecker(t *testing.T) {
	h := newHealth(time.Second, okChecker, failChecker)
	h.SetReady(true)

	code, b := get(t, newMux(h), "/readyz")
	if code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, expected 503", code)
	}
	if b.Status != "fail" {
		t.Errorf("body status = %q, expected fail", b.Status)
	}
	if b.Checks["db"] != "fail" {
		t.Errorf("checks[db] = %q, expected fail", b.Checks["db"])
	}
	if b.Checks["cache"] != "ok" {
		t.Errorf("checks[cache] = %q, expected ok", b.Checks["cache"])
	}
}

func TestReadiness_SlowCheckerTimesOut(t *testing.T) {
	slow := health.NewChecker("slow", func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	})
	h := newHealth(50*time.Millisecond, slow)
	h.SetReady(true)

	start := time.Now()
	code, b := get(t, newMux(h), "/readyz")
	elapsed := time.Since(start)

	if elapsed > time.Second {
		t.Errorf("readiness took %v, expected it to stop near the 50ms timeout", elapsed)
	}
	if code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, expected 503", code)
	}
	if b.Checks["slow"] != "fail" {
		t.Errorf("checks[slow] = %q, expected fail", b.Checks["slow"])
	}
}

func TestReadiness_NotReadyAgainOnShutdown(t *testing.T) {
	h := newHealth(time.Second)
	mux := newMux(h)

	h.SetReady(true)
	if code, _ := get(t, mux, "/readyz"); code != http.StatusOK {
		t.Fatalf("status = %d after SetReady(true), expected 200", code)
	}

	h.SetReady(false)
	if code, _ := get(t, mux, "/readyz"); code != http.StatusServiceUnavailable {
		t.Errorf("status = %d after SetReady(false), expected 503", code)
	}
}

func TestReadiness_Headers(t *testing.T) {
	h := newHealth(time.Second)
	h.SetReady(true)

	rec := httptest.NewRecorder()
	newMux(h).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))

	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, expected application/json", ct)
	}
	if cc := rec.Header().Get("Cache-Control"); cc != "no-store" {
		t.Errorf("Cache-Control = %q, expected no-store", cc)
	}
}

func TestEndpoints_RejectOtherMethods(t *testing.T) {
	mux := newMux(newHealth(time.Second))

	for _, path := range []string{"/healthz", "/readyz"} {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, path, nil))
		if rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("POST %s = %d, expected 405", path, rec.Code)
		}
	}
}
