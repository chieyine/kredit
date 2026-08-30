package observability

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

// Tracer owns the process-level OTLP exporter. Development uses a no-op tracer
// so local requests never block on an unavailable collector.
type Tracer struct {
	provider *sdktrace.TracerProvider
	tracer   trace.Tracer
}

func NewNoopTracer() *Tracer {
	return &Tracer{tracer: otel.Tracer("kredit/noop")}
}

func NewTracer(ctx context.Context, endpoint, serviceName string) (*Tracer, error) {
	if strings.TrimSpace(endpoint) == "" {
		return NewNoopTracer(), nil
	}
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		if err == nil {
			err = fmt.Errorf("OTel endpoint must be an absolute URL")
		}
		return nil, err
	}
	options := []otlptracehttp.Option{otlptracehttp.WithEndpointURL(endpoint), otlptracehttp.WithTimeout(5 * time.Second)}
	if parsed.Scheme == "http" {
		options = append(options, otlptracehttp.WithInsecure())
	}
	exporter, err := otlptracehttp.New(ctx, options...)
	if err != nil {
		return nil, err
	}
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource.NewWithAttributes("https://opentelemetry.io/schemas/1.26.0", attribute.String("service.name", serviceName))),
	)
	return &Tracer{provider: provider, tracer: provider.Tracer("kredit")}, nil
}

func (t *Tracer) Start(ctx context.Context, name string, attributes ...attribute.KeyValue) (context.Context, trace.Span) {
	if t == nil || t.tracer == nil {
		return ctx, trace.SpanFromContext(ctx)
	}
	return t.tracer.Start(ctx, name, trace.WithAttributes(attributes...))
}

func (t *Tracer) Shutdown(ctx context.Context) error {
	if t == nil || t.provider == nil {
		return nil
	}
	return t.provider.Shutdown(ctx)
}

// ExtractTraceContext accepts only W3C trace context and does not propagate
// baggage, which avoids accidentally carrying sensitive values downstream.
func ExtractTraceContext(ctx context.Context, headers map[string]string) context.Context {
	carrier := propagation.MapCarrier{}
	for key, value := range headers {
		if strings.EqualFold(key, "traceparent") {
			carrier["traceparent"] = value
		}
	}
	return propagation.TraceContext{}.Extract(ctx, carrier)
}
