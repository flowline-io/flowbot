package trace

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// Tracer returns a named tracer from the global TracerProvider.
// Use this to create spans in packages that don't need their own tracer instance.
func Tracer(name string) trace.Tracer {
	return otel.Tracer(name)
}

// DetachContext returns a context that keeps values (including the OTel SpanContext)
// from parent but is not canceled when parent is canceled.
// Use for fire-and-forget work that must outlive an HTTP or Watermill handler.
func DetachContext(parent context.Context) context.Context {
	if parent == nil {
		return context.Background()
	}
	return context.WithoutCancel(parent)
}

// DetachWithTimeout returns DetachContext(parent) wrapped in WithTimeout.
func DetachWithTimeout(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(DetachContext(parent), timeout)
}

// StartSpan starts a new span with the given name and returns the updated context and span.
// The span is created as a child of any existing span in ctx.
func StartSpan(ctx context.Context, name string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	return otel.Tracer("flowbot").Start(ctx, name, trace.WithAttributes(attrs...))
}

// SetSpanAttributes sets attributes on the span from the context if one exists.
func SetSpanAttributes(ctx context.Context, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		span.SetAttributes(attrs...)
	}
}

// RecordError records an error on the current span in ctx.
func RecordError(ctx context.Context, err error) {
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		span.RecordError(err)
	}
}

// AddEvent adds an event to the current span in ctx.
func AddEvent(ctx context.Context, name string, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		span.AddEvent(name, trace.WithAttributes(attrs...))
	}
}
