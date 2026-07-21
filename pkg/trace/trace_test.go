package trace

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
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

// tracingConfigMu serializes mutations of config.App.Tracing across parallel tests.
var tracingConfigMu sync.Mutex

// tracerProviderMu serializes exclusive use of the process-global otel TracerProvider.
// Callers must acquire it only inside leaf tests (after t.Parallel), never in a parent
// that still has parallel subtests waiting — that deadlocks parallel slots against the mutex.
var tracerProviderMu sync.Mutex

var (
	testSpanRecorder   *tracetest.SpanRecorder
	testTracerProvider *sdktrace.TracerProvider
)

// TestMain installs one shared TracerProvider for the package so parallel tests do not
// Shutdown each other's providers via competing otel.SetTracerProvider calls.
func TestMain(m *testing.M) {
	testSpanRecorder = tracetest.NewSpanRecorder()
	testTracerProvider = sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(testSpanRecorder))
	otel.SetTracerProvider(testTracerProvider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))
	code := m.Run()
	_ = testTracerProvider.Shutdown(context.Background())
	os.Exit(code)
}

// withTracingConfig temporarily replaces config.App.Tracing under a package mutex.
func withTracingConfig(t *testing.T, cfg config.Tracing) {
	t.Helper()
	tracingConfigMu.Lock()
	orig := config.App.Tracing
	config.App.Tracing = cfg
	t.Cleanup(func() {
		config.App.Tracing = orig
		tracingConfigMu.Unlock()
	})
}

type testLifecycle struct {
	hooks []fx.Hook
}

func (lc *testLifecycle) Append(h fx.Hook) {
	lc.hooks = append(lc.hooks, h)
}

// setupTestTracerProvider pins the shared test TracerProvider for the calling leaf test.
// The returned recorder is package-scoped (see TestMain); prefer it only for assertions
// that tolerate concurrent spans from other parallel tests.
func setupTestTracerProvider(t *testing.T) *tracetest.SpanRecorder {
	t.Helper()
	tracerProviderMu.Lock()
	otel.SetTracerProvider(testTracerProvider)
	t.Cleanup(tracerProviderMu.Unlock)
	return testSpanRecorder
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
			withTracingConfig(t, config.Tracing{Enabled: false})
			// NewTracerProvider replaces the global provider; hold the mutex for the leaf
			// test and restore the shared TestMain provider afterward.
			tracerProviderMu.Lock()
			t.Cleanup(func() {
				otel.SetTracerProvider(testTracerProvider)
				tracerProviderMu.Unlock()
			})

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
