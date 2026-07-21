// Package trace provides OpenTelemetry tracing integration for Fiber.
package trace

import (
	"strconv"

	"github.com/gofiber/fiber/v3"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	oteltrace "go.opentelemetry.io/otel/trace"

	"github.com/flowline-io/flowbot/pkg/utils"
)

// SpanKey is the key used to store the span in fiber.Ctx.Locals().
const SpanKey = "otel_span"

var skippedPaths = []string{
	"/livez",
	"/readyz",
	"/healthz",
	"/metrics",
}

// FiberMiddleware returns a Fiber v3 middleware that creates an OTel span for each request.
//
// The span is named "HTTP {method} {route}" and includes standard HTTP semantic attributes.
// The span is stored in ctx.Locals() under SpanKey for downstream access.
// W3C TraceContext is extracted from incoming headers and propagated.
func FiberMiddleware() fiber.Handler {
	return func(c fiber.Ctx) error {
		if utils.Contains(skippedPaths, c.Path()) {
			return c.Next()
		}

		// Resolve tracer/propagator per request so a late otel.SetTracerProvider
		// (e.g. after app wiring) is observed instead of a stale capture at Use time.
		propagator := otel.GetTextMapPropagator()
		tracer := otel.Tracer("fiber")

		carrier := propagation.HeaderCarrier{}
		for key, value := range c.Request().Header.All() {
			carrier.Set(string(key), string(value))
		}
		ctx := propagator.Extract(c.Context(), carrier)

		method := c.Method()
		path := c.Path()
		spanName := "HTTP " + method + " " + path
		ctx, span := tracer.Start(ctx, spanName,
			oteltrace.WithSpanKind(oteltrace.SpanKindServer),
			oteltrace.WithAttributes(
				semconv.HTTPMethodKey.String(method),
				semconv.HTTPTargetKey.String(string(c.Request().URI().RequestURI())),
				semconv.NetHostNameKey.String(c.Hostname()),
				attribute.String("http.scheme", c.Scheme()),
			),
		)
		defer span.End()

		c.SetContext(ctx)
		c.Locals(SpanKey, span)

		err := c.Next()

		route := path
		if r := c.Route(); r != nil && r.Path != "" {
			route = r.Path
		}
		span.SetName("HTTP " + method + " " + route)
		span.SetAttributes(semconv.HTTPRouteKey.String(route))

		status := c.Response().StatusCode()
		span.SetAttributes(semconv.HTTPStatusCodeKey.Int(status))

		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		} else if status >= 400 {
			span.SetStatus(codes.Error, "HTTP error "+strconv.Itoa(status))
		}

		return err
	}
}

// SpanFromFiber returns the current OTel span from a Fiber context.
func SpanFromFiber(c fiber.Ctx) oteltrace.Span {
	if s, ok := c.Locals(SpanKey).(oteltrace.Span); ok {
		return s
	}
	return oteltrace.SpanFromContext(c.Context())
}
