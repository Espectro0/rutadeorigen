package httpserver

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"rutadeorigen/lib/httpserver/middleware"
)

type Server struct {
	srv *http.Server
	ln  net.Listener
}

func New(addr string, h http.Handler, log *slog.Logger) *Server {
	return &Server{
		srv: &http.Server{
			Addr:              addr,
			Handler:           instrument(h, log),
			ReadHeaderTimeout: 5 * time.Second,
			ReadTimeout:       15 * time.Second,
			WriteTimeout:      15 * time.Second,
			IdleTimeout:       60 * time.Second,
			ErrorLog:          slog.NewLogLogger(log.Handler(), slog.LevelError),
		},
	}
}

func instrument(h http.Handler, log *slog.Logger) http.Handler {
	return otelhttp.NewHandler(
		middleware.Correlation(middleware.AccessLog(log, h)),
		"http.server",
		otelhttp.WithFilter(isNotHealthCheck),
	)
}

func isNotHealthCheck(r *http.Request) bool {
	return r.URL.Path != "/healthz" && r.URL.Path != "/readyz"
}

func (s *Server) Listen() error {
	ln, err := net.Listen("tcp", s.srv.Addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", s.srv.Addr, err)
	}
	s.ln = ln
	return nil
}

func (s *Server) Serve() error {
	if s.ln == nil {
		return errors.New("httpserver: Serve called before Listen")
	}

	err := s.srv.Serve(s.ln)
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}

	return fmt.Errorf("serve: %w", err)
}

func (s *Server) Shutdown(ctx context.Context) error {
	err := s.srv.Shutdown(ctx)
	if err == nil || errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return fmt.Errorf("shutdown: %w", err)
}

func (s *Server) Addr() string {
	if s.ln == nil {
		return s.srv.Addr
	}
	return s.ln.Addr().String()
}
