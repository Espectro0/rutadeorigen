package config_test

import (
	"log/slog"
	"strings"
	"testing"
	"time"

	"rutadeorigen/lib/config"
)

func setEnv(t *testing.T, kv map[string]string) {
	t.Helper()
	for _, k := range []string{"SERVICE_NAME", "PORT", "LOG_LEVEL", "SHUTDOWN_TIMEOUT", "SHUTDOWN_DELAY"} {
		t.Setenv(k, kv[k])
	}
}

func TestLoadBase_Defaults(t *testing.T) {
	setEnv(t, map[string]string{"SERVICE_NAME": "qr"})

	b, err := config.LoadBase()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if b.Port != "8080" {
		t.Errorf("Port = %q, expected 8080", b.Port)
	}
	if b.LogLevel != slog.LevelInfo {
		t.Errorf("LogLevel = %v, expected info", b.LogLevel)
	}

	if b.ShutdownTimeout != 10*time.Second {
		t.Errorf("ShutdownTimeout = %q, expected 10s", b.ShutdownTimeout)
	}

	if b.ShutdownDelay != 5*time.Second {
		t.Errorf("ShutdownDelay = %q, expected 5s", b.ShutdownDelay)
	}

	if b.Addr() != ":8080" {
		t.Errorf("b.Addr() = %s, expected :8080", b.Addr())
	}

	_ = time.Second
}

func TestLoadBase_Errores(t *testing.T) {
	cases := []struct {
		name string
		env  map[string]string
	}{
		{"Without Service Name", map[string]string{}},
		{"Not number port", map[string]string{"SERVICE_NAME": "qr", "PORT": "abc"}},
		{"Port 0", map[string]string{"SERVICE_NAME": "qr", "PORT": "0"}},
		{"Port 70000", map[string]string{"SERVICE_NAME": "qr", "PORT": "70000"}},
		{"LogLevel Verbose", map[string]string{"SERVICE_NAME": "qr", "LOG_LEVEL": "verbose"}},
		{"Zero ShutDown Timeout", map[string]string{"SERVICE_NAME": "qr", "PORT": "8080", "SHUTDOWN_TIMEOUT": "0"}},
		{"Ten ShutDown Timeout", map[string]string{"SERVICE_NAME": "qr", "PORT": "8080", "SHUTDOWN_TIMEOUT": "ten"}},
		{"Negative ShutDown Delay", map[string]string{"SERVICE_NAME": "qr", "PORT": "8080", "SHUTDOWN_DELAY": "-1"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			setEnv(t, c.env)
			if _, err := config.LoadBase(); err == nil {
				t.Fatal("Expected Error")
			}
		})
	}
}

func TestLoadBase_Overrides(t *testing.T) {
	setEnv(t, map[string]string{
		"SERVICE_NAME":     "traceability",
		"PORT":             "9090",
		"LOG_LEVEL":        "debug",
		"SHUTDOWN_TIMEOUT": "30s",
		"SHUTDOWN_DELAY":   "0s",
	})

	b, err := config.LoadBase()
	if err != nil {
		t.Fatalf("Unexpected Error: %v", err)
	}

	if b.ServiceName != "traceability" {
		t.Errorf("ServiceName = %q, expected traceability", b.ServiceName)
	}
	if b.Port != "9090" {
		t.Errorf("Port = %q, expected 9090", b.Port)
	}
	if b.Addr() != ":9090" {
		t.Errorf("Addr() = %q, expected :9090", b.Addr())
	}
	if b.LogLevel != slog.LevelDebug {
		t.Errorf("LogLevel = %v, expected debug", b.LogLevel)
	}
	if b.ShutdownTimeout != 30*time.Second {
		t.Errorf("ShutdownTimeout = %v, expected 30s", b.ShutdownTimeout)
	}
	if b.ShutdownDelay != 0 {
		t.Errorf("ShutdownDelay = %v, expected 0s", b.ShutdownDelay)
	}
}

func TestLoadBase_VariosErrores(t *testing.T) {
	setEnv(t, map[string]string{
		"PORT": "abc",
	})

	_, err := config.LoadBase()
	if err == nil {
		t.Fatal("Unexpected Error")
	}

	msg := err.Error()
	for _, clave := range []string{"SERVICE_NAME", "PORT"} {
		if !strings.Contains(msg, clave) {
			t.Errorf("Error %s; complete message:\n%s", clave, msg)
		}
	}
}
