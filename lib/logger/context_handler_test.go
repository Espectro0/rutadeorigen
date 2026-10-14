package logger_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"

	"rutadeorigen/lib/logger"
	"rutadeorigen/lib/requestcontext"
)

var (
	sampleTraceID = trace.TraceID{0x4b, 0xf9, 0x2f, 0x35, 0x77, 0xb3, 0x4d, 0xa6, 0xa3, 0xce, 0x92, 0x9d, 0x0e, 0x0e, 0x47, 0x36}
	sampleSpanID  = trace.SpanID{0x00, 0xf0, 0x67, 0xaa, 0x0b, 0xa9, 0x02, 0xb7}
)

func decodeEntry(t *testing.T, buf *bytes.Buffer) map[string]any {
	t.Helper()

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, buf.String())
	}
	return entry
}

func fullContext(actorID uuid.UUID) context.Context {
	ctx := requestcontext.WithCorrelationID(context.Background(), "req-123")
	ctx = requestcontext.WithActorID(ctx, actorID)
	return trace.ContextWithSpanContext(ctx, trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    sampleTraceID,
		SpanID:     sampleSpanID,
		TraceFlags: trace.FlagsSampled,
	}))
}

func TestContextHandler_AddsRequestFields(t *testing.T) {
	var buf bytes.Buffer
	actorID := uuid.New()

	logger.New(&buf, "traceability", slog.LevelInfo).InfoContext(fullContext(actorID), "lot created")

	entry := decodeEntry(t, &buf)
	want := map[string]string{
		"service":        "traceability",
		"msg":            "lot created",
		"correlation_id": "req-123",
		"actor_id":       actorID.String(),
		"trace_id":       "4bf92f3577b34da6a3ce929d0e0e4736",
		"span_id":        "00f067aa0ba902b7",
	}
	for k, v := range want {
		if entry[k] != v {
			t.Errorf("%s = %v, expected %q", k, entry[k], v)
		}
	}
}

func TestContextHandler_EmptyContextAddsNothing(t *testing.T) {
	var buf bytes.Buffer

	logger.New(&buf, "traceability", slog.LevelInfo).InfoContext(context.Background(), "started")

	entry := decodeEntry(t, &buf)
	for _, k := range []string{"correlation_id", "actor_id", "trace_id", "span_id"} {
		if _, ok := entry[k]; ok {
			t.Errorf("%s must be omitted when the context does not carry it, got %v", k, entry[k])
		}
	}
}

func TestContextHandler_IgnoresInvalidSpanContext(t *testing.T) {
	var buf bytes.Buffer
	ctx := trace.ContextWithSpanContext(context.Background(), trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: sampleTraceID,
	}))

	logger.New(&buf, "traceability", slog.LevelInfo).InfoContext(ctx, "no span id")

	entry := decodeEntry(t, &buf)
	if _, ok := entry["trace_id"]; ok {
		t.Errorf("trace_id must be omitted for an invalid span context, got %v", entry["trace_id"])
	}
}

func TestContextHandler_KeepsFieldsAfterWith(t *testing.T) {
	var buf bytes.Buffer

	log := logger.New(&buf, "traceability", slog.LevelInfo).With("component", "lots")
	log.InfoContext(fullContext(uuid.New()), "with attrs")

	entry := decodeEntry(t, &buf)
	if entry["component"] != "lots" {
		t.Errorf("component = %v, expected lots", entry["component"])
	}
	if entry["correlation_id"] != "req-123" || entry["trace_id"] == nil {
		t.Errorf("context fields lost after With: %v", entry)
	}
}

func TestContextHandler_KeepsFieldsAfterWithGroup(t *testing.T) {
	var buf bytes.Buffer

	log := logger.New(&buf, "traceability", slog.LevelInfo).WithGroup("request")
	log.InfoContext(fullContext(uuid.New()), "with group")

	entry := decodeEntry(t, &buf)
	group, ok := entry["request"].(map[string]any)
	if !ok {
		t.Fatalf("expected the request group, got %v", entry)
	}
	if group["correlation_id"] != "req-123" {
		t.Errorf("correlation_id = %v, expected req-123 inside the group", group["correlation_id"])
	}
}

func TestContextHandler_RespectsLevel(t *testing.T) {
	var buf bytes.Buffer

	logger.New(&buf, "traceability", slog.LevelWarn).InfoContext(fullContext(uuid.New()), "filtered")

	if buf.Len() != 0 {
		t.Errorf("info must be filtered with level warn, got %s", buf.String())
	}
}
