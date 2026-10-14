package httpserver_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"rutadeorigen/lib/httpserver"
	"rutadeorigen/lib/httpserver/middleware"
	"rutadeorigen/lib/logger"
	"rutadeorigen/lib/telemetry"
)

const (
	incomingTraceID = "4bf92f3577b34da6a3ce929d0e0e4736"
	incomingParent  = "00-" + incomingTraceID + "-00f067aa0ba902b7-01"
)

type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) entries(t *testing.T) []map[string]any {
	t.Helper()
	b.mu.Lock()
	defer b.mu.Unlock()

	var entries []map[string]any
	for _, line := range strings.Split(strings.TrimSpace(b.buf.String()), "\n") {
		if line == "" {
			continue
		}
		var entry map[string]any
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			t.Fatalf("log line is not valid JSON: %v\n%s", err, line)
		}
		entries = append(entries, entry)
	}
	return entries
}

func findEntry(entries []map[string]any, msg string) map[string]any {
	for _, e := range entries {
		if e["msg"] == msg {
			return e
		}
	}
	return nil
}

func startInstrumentedServer(t *testing.T, logs *syncBuffer) string {
	t.Helper()
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "")
	t.Setenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", "")

	shutdownTelemetry, err := telemetry.Init(context.Background(), "test", "dev")
	if err != nil {
		t.Fatalf("telemetry.Init: %v", err)
	}

	log := logger.New(logs, "test", slog.LevelInfo)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /lots/{id}", func(w http.ResponseWriter, r *http.Request) {
		log.InfoContext(r.Context(), "handler")
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		log.InfoContext(r.Context(), "health")
		w.WriteHeader(http.StatusOK)
	})

	srv := httpserver.New("127.0.0.1:0", mux, log)
	if err := srv.Listen(); err != nil {
		t.Fatalf("Listen: %v", err)
	}
	go func() { _ = srv.Serve() }()

	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
		_ = shutdownTelemetry(ctx)
	})

	return "http://" + srv.Addr()
}

func get(t *testing.T, url string, headers map[string]string) *http.Response {
	t.Helper()

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	_ = resp.Body.Close()
	return resp
}

func TestServer_LogsCarryTraceAndCorrelation(t *testing.T) {
	var logs syncBuffer
	base := startInstrumentedServer(t, &logs)

	resp := get(t, base+"/lots/42", map[string]string{
		"traceparent":                  incomingParent,
		middleware.CorrelationIDHeader: "req-xyz",
	})
	if resp.Header.Get(middleware.CorrelationIDHeader) != "req-xyz" {
		t.Errorf("correlation id must be echoed, got %q", resp.Header.Get(middleware.CorrelationIDHeader))
	}

	entries := logs.entries(t)
	for _, msg := range []string{"handler", "http request"} {
		entry := findEntry(entries, msg)
		if entry == nil {
			t.Fatalf("missing %q log line in %v", msg, entries)
		}
		if entry["trace_id"] != incomingTraceID {
			t.Errorf("%s: trace_id = %v, expected the incoming %s", msg, entry["trace_id"], incomingTraceID)
		}
		if entry["span_id"] == nil || entry["span_id"] == "00f067aa0ba902b7" {
			t.Errorf("%s: span_id must be the server span, got %v", msg, entry["span_id"])
		}
		if entry["correlation_id"] != "req-xyz" {
			t.Errorf("%s: correlation_id = %v, expected req-xyz", msg, entry["correlation_id"])
		}
	}

	access := findEntry(entries, "http request")
	if access["http.route"] != "GET /lots/{id}" {
		t.Errorf("http.route = %v, expected GET /lots/{id}", access["http.route"])
	}
}

func TestServer_StartsNewTraceWithoutTraceparent(t *testing.T) {
	var logs syncBuffer
	base := startInstrumentedServer(t, &logs)

	get(t, base+"/lots/1", nil)

	entry := findEntry(logs.entries(t), "handler")
	if entry == nil {
		t.Fatal("missing handler log line")
	}
	traceID, ok := entry["trace_id"].(string)
	if !ok || len(traceID) != 32 || traceID == incomingTraceID {
		t.Errorf("expected a new trace id, got %v", entry["trace_id"])
	}
}

func TestServer_HealthChecksAreNotTraced(t *testing.T) {
	var logs syncBuffer
	base := startInstrumentedServer(t, &logs)

	get(t, base+"/healthz", map[string]string{"traceparent": incomingParent})

	entries := logs.entries(t)
	if findEntry(entries, "http request") != nil {
		t.Error("health checks must not produce an access log line")
	}
	entry := findEntry(entries, "health")
	if entry == nil {
		t.Fatal("missing health log line")
	}
	if _, ok := entry["trace_id"]; ok {
		t.Errorf("health checks must not start a span, got trace_id %v", entry["trace_id"])
	}
}
