package logger

import (
	"io"
	"log/slog"
)

func New(w io.Writer, service string, level slog.Level) *slog.Logger {
	handler := slog.NewJSONHandler(w, &slog.HandlerOptions{Level: level})

	return slog.New(handler).With("service", service)
}
