package logger_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"testing"

	"rutadeorigen/lib/logger"
)

func TestNew_WritesJSONWithService(t *testing.T) {
	var buf bytes.Buffer
	log := logger.New(&buf, "qr", slog.LevelInfo)

	log.Info("server started", "port", "8080")

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, buf.String())
	}

	want := map[string]string{
		"service": "qr",
		"msg":     "server started",
		"level":   "INFO",
		"port":    "8080",
	}
	for k, v := range want {
		if entry[k] != v {
			t.Errorf("%s = %v, expected %q", k, entry[k], v)
		}
	}

	if _, ok := entry["time"]; !ok {
		t.Error("time key is missing")
	}
}

func TestNew_FiltersByLevel(t *testing.T) {
	var buf bytes.Buffer
	log := logger.New(&buf, "qr", slog.LevelInfo)

	log.Debug("should not be written")
	if buf.Len() != 0 {
		t.Fatalf("debug was written with level info:\n%s", buf.String())
	}

	log.Warn("should be written")
	if buf.Len() == 0 {
		t.Fatal("warn was not written with level info")
	}
}

func TestNew_DebugLevel(t *testing.T) {
	var buf bytes.Buffer
	log := logger.New(&buf, "qr", slog.LevelDebug)

	log.Debug("should be written")
	if buf.Len() == 0 {
		t.Fatal("debug was not written with level debug")
	}
}
