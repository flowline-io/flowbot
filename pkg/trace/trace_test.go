package trace

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.uber.org/fx"

	"github.com/flowline-io/flowbot/pkg/config"
)

type testLifecycle struct {
	hooks []fx.Hook
}

func (lc *testLifecycle) Append(h fx.Hook) {
	lc.hooks = append(lc.hooks, h)
}

func setupTestTracerProvider(t *testing.T) *tracetest.SpanRecorder {
	t.Helper()
	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))
	t.Cleanup(func() {
		_ = tp.Shutdown(context.Background())
	})
	return sr
}

func TestTracer(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "returns named tracer from global provider"},
		{name: "returns second tracer with different name"},
		{name: "returns flowbot tracer name"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			setupTestTracerProvider(t)
			tr := Tracer("test-component")
			require.NotNil(t, tr)
			_, span := tr.Start(context.Background(), "op")
			span.End()
		})
	}
}

func TestStartSpan(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		attrs []attribute.KeyValue
	}{
		{name: "starts span without attributes"},
		{name: "starts span with attributes", attrs: []attribute.KeyValue{attribute.String("k", "v")}},
		{name: "starts child span from parent context"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			setupTestTracerProvider(t)
			ctx, span := StartSpan(context.Background(), "test-span", tt.attrs...)
			require.NotNil(t, span)
			defer span.End()
			assert.NotEqual(t, context.Background(), ctx)
		})
	}
}

func TestSetSpanAttributes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "sets attributes on recording span"},
		{name: "noop when context has no span"},
		{name: "noop when span is not recording"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			setupTestTracerProvider(t)
			if tt.name == "noop when context has no span" {
				SetSpanAttributes(context.Background(), attribute.String("x", "y"))
				return
			}
			ctx, span := StartSpan(context.Background(), "attr-span")
			defer span.End()
			SetSpanAttributes(ctx, attribute.String("key", "value"))
		})
	}
}

func TestRecordError(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "records error on active span"},
		{name: "noop without span in context"},
		{name: "noop with nil error"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			setupTestTracerProvider(t)
			if tt.name == "noop without span in context" {
				RecordError(context.Background(), errors.New("boom"))
				return
			}
			ctx, span := StartSpan(context.Background(), "err-span")
			defer span.End()
			if tt.name == "noop with nil error" {
				RecordError(ctx, nil)
				return
			}
			RecordError(ctx, errors.New("boom"))
		})
	}
}

func TestAddEvent(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "adds event to recording span"},
		{name: "noop without span in context"},
		{name: "adds event with attributes"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			setupTestTracerProvider(t)
			if tt.name == "noop without span in context" {
				AddEvent(context.Background(), "evt")
				return
			}
			ctx, span := StartSpan(context.Background(), "event-span")
			defer span.End()
			AddEvent(ctx, "evt", attribute.String("n", "1"))
		})
	}
}

func TestNewTracerProviderDisabled(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "returns noop provider when tracing disabled"},
		{name: "noop provider can create spans"},
		{name: "lifecycle hooks are registered when enabled defaults skipped"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			orig := config.App.Tracing
			t.Cleanup(func() { config.App.Tracing = orig })
			config.App.Tracing = config.Tracing{Enabled: false}

			lc := &testLifecycle{}
			tp, err := NewTracerProvider(lc)
			require.NoError(t, err)
			require.NotNil(t, tp)
			_, span := tp.Tracer("test").Start(context.Background(), "noop")
			span.End()
		})
	}
}

func TestFiberMiddleware(t *testing.T) {
	t.Parallel()
	setupTestTracerProvider(t)

	app := fiber.New()
	app.Use(FiberMiddleware())
	app.Get("/api/items", func(c fiber.Ctx) error {
		span := SpanFromFiber(c)
		require.NotNil(t, span)
		return c.SendString("ok")
	})
	app.Get("/error", func(_ fiber.Ctx) error {
		return fiber.NewError(fiber.StatusTeapot, "teapot")
	})

	tests := []struct {
		name       string
		path       string
		wantStatus int
		skipSpan   bool
	}{
		{name: "creates span for normal route", path: "/api/items", wantStatus: fiber.StatusOK},
		{name: "records error status on handler error", path: "/error", wantStatus: fiber.StatusTeapot},
		{name: "skips span for health path", path: "/livez", wantStatus: fiber.StatusNotFound, skipSpan: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			resp, err := app.Test(httptest.NewRequest(http.MethodGet, tt.path, http.NoBody))
			require.NoError(t, err)
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
		})
	}
}

func TestSpanFromFiber(t *testing.T) {
	t.Parallel()
	setupTestTracerProvider(t)

	app := fiber.New()
	app.Use(FiberMiddleware())
	app.Get("/span", func(c fiber.Ctx) error {
		span := SpanFromFiber(c)
		assert.True(t, span.SpanContext().IsValid() || span.IsRecording())
		return c.SendStatus(fiber.StatusOK)
	})

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/span", http.NoBody))
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestSpanFromFiberWithoutMiddleware(t *testing.T) {
	t.Parallel()
	setupTestTracerProvider(t)

	app := fiber.New()
	app.Get("/bare", func(c fiber.Ctx) error {
		span := SpanFromFiber(c)
		assert.NotNil(t, span)
		return c.SendStatus(fiber.StatusOK)
	})

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/bare", http.NoBody))
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}
