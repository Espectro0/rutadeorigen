package telemetry_test

import (
	"context"
	"testing"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"rutadeorigen/lib/telemetry"
)

func withoutExporter(t *testing.T) {
	t.Helper()
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "")
	t.Setenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", "")
}

func initTelemetry(t *testing.T, service, version string) {
	t.Helper()

	shutdown, err := telemetry.Init(context.Background(), service, version)
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := shutdown(ctx); err != nil {
			t.Errorf("shutdown: %v", err)
		}
	})
}

func TestInit_CreatesValidSpansWithoutExporter(t *testing.T) {
	withoutExporter(t)
	initTelemetry(t, "traceability", "v1.2.3")

	_, span := otel.Tracer("test").Start(context.Background(), "operation")
	defer span.End()

	if !span.SpanContext().IsValid() {
		t.Fatal("spans must have a valid trace and span id even without an exporter")
	}
	if !span.SpanContext().IsSampled() {
		t.Error("root spans must be sampled")
	}
}

func TestInit_SetsServiceResource(t *testing.T) {
	withoutExporter(t)
	initTelemetry(t, "traceability", "v1.2.3")

	_, span := otel.Tracer("test").Start(context.Background(), "operation")
	defer span.End()

	readOnly, ok := span.(sdktrace.ReadOnlySpan)
	if !ok {
		t.Fatalf("expected an SDK span, got %T", span)
	}

	attrs := map[string]string{}
	for _, kv := range readOnly.Resource().Attributes() {
		attrs[string(kv.Key)] = kv.Value.String()
	}
	if attrs["service.name"] != "traceability" {
		t.Errorf("service.name = %q, expected traceability", attrs["service.name"])
	}
	if attrs["service.version"] != "v1.2.3" {
		t.Errorf("service.version = %q, expected v1.2.3", attrs["service.version"])
	}
}

func TestInit_PropagatesW3CTraceContext(t *testing.T) {
	withoutExporter(t)
	initTelemetry(t, "traceability", "dev")

	ctx, span := otel.Tracer("test").Start(context.Background(), "outgoing")
	defer span.End()

	carrier := propagation.MapCarrier{}
	otel.GetTextMapPropagator().Inject(ctx, carrier)

	want := "00-" + span.SpanContext().TraceID().String() + "-" + span.SpanContext().SpanID().String() + "-01"
	if got := carrier.Get("traceparent"); got != want {
		t.Errorf("traceparent = %q, expected %q", got, want)
	}

	extracted := otel.GetTextMapPropagator().Extract(context.Background(), carrier)
	_, child := otel.Tracer("test").Start(extracted, "incoming")
	defer child.End()

	if child.SpanContext().TraceID() != span.SpanContext().TraceID() {
		t.Error("a span started from an extracted context must keep the same trace id")
	}
}

func TestInit_WithExporterEndpoint(t *testing.T) {
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://127.0.0.1:4318")
	t.Setenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", "")

	initTelemetry(t, "traceability", "dev")
}
