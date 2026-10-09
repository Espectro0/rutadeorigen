package httpserver_test

import (
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
	"testing"
	"time"

	"rutadeorigen/lib/httpserver"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

func TestServer_ServesAndShutsDown(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /ping", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("pong"))
	})

	srv := httpserver.New("127.0.0.1:0", mux, discardLogger())
	if err := srv.Listen(); err != nil {
		t.Fatalf("Listen: %v", err)
	}

	errCh := make(chan error, 1)
	go func() { errCh <- srv.Serve() }()

	resp, err := http.Get("http://" + srv.Addr() + "/ping")
	if err != nil {
		t.Fatalf("GET /ping: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, expected 200", resp.StatusCode)
	}
	if string(body) != "pong" {
		t.Errorf("body = %q, expected pong", body)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}

	select {
	case err := <-errCh:
		if err != nil {
			t.Errorf("Serve returned %v after Shutdown, expected nil", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Serve did not return after Shutdown")
	}
}

func TestServer_ListenFailsWhenPortIsTaken(t *testing.T) {
	taken, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	defer taken.Close()

	srv := httpserver.New(taken.Addr().String(), http.NewServeMux(), discardLogger())
	if err := srv.Listen(); err == nil {
		t.Fatal("expected error when port is already in use")
	}
}

func TestServer_ServeBeforeListen(t *testing.T) {
	srv := httpserver.New("127.0.0.1:0", http.NewServeMux(), discardLogger())
	if err := srv.Serve(); err == nil {
		t.Fatal("expected error when Serve is called before Listen")
	}
}

func TestServer_AddrBeforeAndAfterListen(t *testing.T) {
	srv := httpserver.New("127.0.0.1:0", http.NewServeMux(), discardLogger())

	if got := srv.Addr(); got != "127.0.0.1:0" {
		t.Errorf("Addr before Listen = %q, expected 127.0.0.1:0", got)
	}

	if err := srv.Listen(); err != nil {
		t.Fatalf("Listen: %v", err)
	}
	t.Cleanup(func() { _ = srv.Shutdown(context.Background()) })

	if got := srv.Addr(); got == "127.0.0.1:0" {
		t.Errorf("Addr after Listen = %q, expected a real port", got)
	}
}
