package trace

import (
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/samber/lo"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// SpanKey is the key used to store the span in fiber.Ctx.Locals().
const SpanKey = "otel_span"

var skippedPaths = []string{
	"/livez",
	"/readyz",
	"/healthz",
	"/metrics",
	"/server-debugger/debug/trace",
}

// FiberMiddleware returns a Fiber v3 middleware that creates an OTel span for each request.
//
// The span is named "HTTP {method} {route}" and includes standard HTTP semantic attributes.
// The span is stored in ctx.Locals() under SpanKey for downstream access.
// W3C TraceContext is extracted from incoming headers and propagated.
func FiberMiddleware() fiber.Handler {
	propagator := otel.GetTextMapPropagator()
	tracer := otel.Tracer("fiber")

	return func(c fiber.Ctx) error {
		if lo.Contains(skippedPaths, c.Path()) {
			return c.Next()
		}

		carrier := propagation.HeaderCarrier{}
		c.Request().Header.VisitAll(func(key, value []byte) {
			carrier.Set(string(key), string(value))
		})
		ctx := propagator.Extract(c.Context(), carrier)

		method := c.Method()
		route := ""
		if r := c.Route(); r != nil {
			route = r.Path
		}
		if route == "" {
			route = c.Path()
		}

		spanName := "HTTP " + method + " " + route
		ctx, span := tracer.Start(ctx, spanName,
			oteltrace.WithSpanKind(oteltrace.SpanKindServer),
			oteltrace.WithAttributes(
				semconv.HTTPMethodKey.String(method),
				semconv.HTTPRouteKey.String(route),
				semconv.HTTPTargetKey.String(string(c.Request().URI().RequestURI())),
				semconv.NetHostNameKey.String(c.Hostname()),
				attribute.String("http.scheme", c.Scheme()),
			),
		)
		defer span.End()

		c.SetContext(ctx)
		c.Locals(SpanKey, span)

		err := c.Next()

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
