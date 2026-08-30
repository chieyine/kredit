package observability

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel/trace"
)

func TestNoopTracerStartsSafeSpan(t *testing.T) {
	tracer := NewNoopTracer()
	ctx, span := tracer.Start(context.Background(), "test")
	if ctx == nil || span == nil {
		t.Fatal("expected a usable no-op span")
	}
	span.End()
}

func TestExtractTraceContextOnlyUsesTraceparent(t *testing.T) {
	ctx := ExtractTraceContext(context.Background(), map[string]string{"traceparent": "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01", "baggage": "phone=redacted"})
	span := trace.SpanContextFromContext(ctx)
	if !span.IsValid() || span.TraceID().String() != "4bf92f3577b34da6a3ce929d0e0e4736" {
		t.Fatalf("trace context was not extracted: %v", span)
	}
}
