package logger_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
	"time"

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

func TestNew_WritesTimesInUTC(t *testing.T) {
	original := time.Local
	time.Local = time.FixedZone("COT", -5*60*60)
	t.Cleanup(func() { time.Local = original })

	var buf bytes.Buffer
	occurredAt := time.Date(2026, time.October, 13, 20, 0, 0, 0, time.Local)

	logger.New(&buf, "qr", slog.LevelInfo).Info("lot created", "occurred_at", occurredAt)

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, buf.String())
	}

	if ts, _ := entry["time"].(string); !strings.HasSuffix(ts, "Z") {
		t.Errorf("time = %q, expected UTC (suffix Z)", ts)
	}
	if entry["occurred_at"] != "2026-10-14T01:00:00Z" {
		t.Errorf("occurred_at = %v, expected 2026-10-14T01:00:00Z", entry["occurred_at"])
	}
}

func TestNew_KeepsNonTimeAttributes(t *testing.T) {
	var buf bytes.Buffer

	logger.New(&buf, "qr", slog.LevelInfo).Info("msg", "count", 3, "date", "2026-10-13")

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, buf.String())
	}
	if entry["count"] != float64(3) || entry["date"] != "2026-10-13" {
		t.Errorf("non-time attributes must not change, got %v", entry)
	}
}
