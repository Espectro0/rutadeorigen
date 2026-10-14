package logger

import (
	"io"
	"log/slog"
)

func New(w io.Writer, service string, level slog.Level) *slog.Logger {
	handler := slog.NewJSONHandler(w, &slog.HandlerOptions{
		Level:       level,
		ReplaceAttr: utcTime,
	})

	return slog.New(contextHandler{handler}).With("service", service)
}

func utcTime(_ []string, a slog.Attr) slog.Attr {
	if a.Value.Kind() == slog.KindTime {
		a.Value = slog.TimeValue(a.Value.Time().UTC())
	}

	return a
}
