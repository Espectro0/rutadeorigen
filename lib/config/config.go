package config

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"
)

type Base struct {
	ServiceName     string
	Port            string
	LogLevel        slog.Level
	ShutdownTimeout time.Duration
	ShutdownDelay   time.Duration
}

func (b Base) Addr() string {
	return ":" + b.Port
}

func LoadBase() (Base, error) {
	var errs []error

	b := Base{
		ServiceName: String("SERVICE_NAME", ""),
		Port:        String("PORT", "8080"),
	}

	if b.ServiceName == "" {
		errs = append(errs, errors.New("SERVICE_NAME is required"))
	}

	var err error

	if b.Port, err = validatePort(b.Port); err != nil {
		errs = append(errs, err)
	}

	if b.LogLevel, err = Level("LOG_LEVEL", slog.LevelInfo); err != nil {
		errs = append(errs, err)
	}

	if b.ShutdownTimeout, err = Duration("SHUTDOWN_TIMEOUT", 10*time.Second); err != nil {
		errs = append(errs, err)
	} else if b.ShutdownTimeout <= 0 {
		errs = append(errs, errors.New("SHUTDOWN_TIMEOUT must be greater than zero"))
	}

	if b.ShutdownDelay, err = Duration("SHUTDOWN_DELAY", 5*time.Second); err != nil {
		errs = append(errs, err)
	} else if b.ShutdownDelay < 0 {
		errs = append(errs, errors.New("SHUTDOWN_DELAY must be greater than zero"))
	}

	if joined := errors.Join(errs...); joined != nil {
		return Base{}, joined
	}

	return b, nil
}

func String(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}

func Duration(key string, def time.Duration) (time.Duration, error) {
	v := String(key, "")
	if v == "" {
		return def, nil
	}

	time, err := time.ParseDuration(v)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", key, err)
	}

	return time, nil
}

func Level(key string, def slog.Level) (slog.Level, error) {
	v := String(key, "")
	if v == "" {
		return def, nil
	}
	var l slog.Level
	if err := l.UnmarshalText([]byte(v)); err != nil {
		return def, fmt.Errorf("%s: %w", key, err)
	}
	return l, nil
}

func validatePort(p string) (string, error) {
	port, err := strconv.Atoi(p)
	if err != nil {
		return "", fmt.Errorf("PORT: %q is not a number %w", p, err)
	}

	if port < 1 || port > 65535 {
		return "", fmt.Errorf("PORT: %d out of range", port)
	}

	return strconv.Itoa(port), nil
}
