package app_test

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	"rutadeorigen/lib/app"
	"rutadeorigen/lib/config"
	"rutadeorigen/lib/health"
)

func baseConfig(port string) config.Base {
	return config.Base{
		ServiceName:     "test",
		Port:            port,
		LogLevel:        slog.LevelInfo,
		ShutdownTimeout: 2 * time.Second,
		ShutdownDelay:   0,
	}
}

func setup() (*http.ServeMux, *health.Health, *slog.Logger) {
	log := slog.New(slog.DiscardHandler)
	h := health.New(log, time.Second)
	mux := http.NewServeMux()
	h.Register(mux)
	return mux, h, log
}

func readyStatus(mux http.Handler) int {
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	return rec.Code
}

func waitReady(t *testing.T, mux http.Handler) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if readyStatus(mux) == http.StatusOK {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("service never became ready")
}

func start(t *testing.T, p app.Params) (context.CancelFunc, <-chan error) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	done := make(chan error, 1)
	go func() { done <- app.Run(ctx, p) }()
	return cancel, done
}

func waitDone(t *testing.T, done <-chan error) error {
	t.Helper()
	select {
	case err := <-done:
		return err
	case <-time.After(3 * time.Second):
		t.Fatal("Run did not return")
		return nil
	}
}

func TestRun_ShutsDownCleanly(t *testing.T) {
	mux, h, log := setup()
	cancel, done := start(t, app.Params{
		Config: baseConfig("0"), Logger: log, Handler: mux, Health: h,
	})

	waitReady(t, mux)
	cancel()

	if err := waitDone(t, done); err != nil {
		t.Errorf("Run returned %v, expected nil", err)
	}
	if code := readyStatus(mux); code != http.StatusServiceUnavailable {
		t.Errorf("readyz = %d after shutdown, expected 503", code)
	}
}

func TestRun_FailsWhenPortIsTaken(t *testing.T) {
	taken, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	defer taken.Close()
	_, port, _ := net.SplitHostPort(taken.Addr().String())

	mux, h, log := setup()
	_, done := start(t, app.Params{
		Config: baseConfig(port), Logger: log, Handler: mux, Health: h,
	})

	if err := waitDone(t, done); err == nil {
		t.Fatal("expected error when port is already in use")
	}
	if code := readyStatus(mux); code != http.StatusServiceUnavailable {
		t.Errorf("readyz = %d, expected 503: service must never be ready", code)
	}
}

func TestRun_ClosesResourcesInReverseOrder(t *testing.T) {
	var order []string
	closer := func(name string) app.Closer {
		return func(context.Context) error {
			order = append(order, name)
			return nil
		}
	}

	mux, h, log := setup()
	cancel, done := start(t, app.Params{
		Config: baseConfig("0"), Logger: log, Handler: mux, Health: h,
		Closers: []app.Closer{closer("db"), closer("broker")},
	})

	waitReady(t, mux)
	cancel()
	if err := waitDone(t, done); err != nil {
		t.Fatalf("Run returned %v, expected nil", err)
	}

	if want := []string{"broker", "db"}; !slices.Equal(order, want) {
		t.Errorf("close order = %v, expected %v", order, want)
	}
}

func TestRun_ReturnsCloserErrors(t *testing.T) {
	errBoom := errors.New("boom")

	mux, h, log := setup()
	cancel, done := start(t, app.Params{
		Config: baseConfig("0"), Logger: log, Handler: mux, Health: h,
		Closers: []app.Closer{func(context.Context) error { return errBoom }},
	})

	waitReady(t, mux)
	cancel()

	if err := waitDone(t, done); !errors.Is(err, errBoom) {
		t.Errorf("Run returned %v, expected it to wrap %v", err, errBoom)
	}
}
