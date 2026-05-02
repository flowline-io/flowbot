package trace

import (
	"context"
	"fmt"
	"time"

	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/flog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.uber.org/fx"
)

// NewTracerProvider creates and configures the global OpenTelemetry TracerProvider.
// It sets up an OTLP HTTP exporter and registers a shutdown hook via fx lifecycle.
func NewTracerProvider(lc fx.Lifecycle) (*sdktrace.TracerProvider, error) {
	cfg := config.App.Tracing
	if !cfg.Enabled {
		flog.Info("tracing disabled, using noop TracerProvider")
		noop := sdktrace.NewTracerProvider()
		otel.SetTracerProvider(noop)
		return noop, nil
	}

	endpoint := cfg.Endpoint
	if endpoint == "" {
		endpoint = "http://localhost:4318/v1/traces"
	}
	serviceName := cfg.ServiceName
	if serviceName == "" {
		serviceName = "flowbot"
	}
	environment := cfg.Environment
	if environment == "" {
		environment = "development"
	}
	sampleRate := cfg.SampleRate
	if sampleRate <= 0 {
		sampleRate = 1.0
	}

	exp, err := otlptracehttp.New(
		context.Background(),
		otlptracehttp.WithEndpointURL(endpoint),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP trace exporter: %w", err)
	}

	res, err := resource.New(
		context.Background(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(serviceName),
			semconv.DeploymentEnvironmentKey.String(environment),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create trace resource: %w", err)
	}

	sp := sdktrace.NewBatchSpanProcessor(exp,
		sdktrace.WithBatchTimeout(5*time.Second),
	)

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(sp),
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(sampleRate))),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			flog.Info("tracing exporter started: endpoint=%s service=%s env=%s sample=%.2f",
				endpoint, serviceName, environment, sampleRate)
			return nil
		},
		OnStop: func(ctx context.Context) error {
			ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
			if err := tp.Shutdown(ctx); err != nil {
				flog.Error(fmt.Errorf("tracing shutdown error: %w", err))
			} else {
				flog.Info("tracing exporter shut down")
			}
			return nil
		},
	})

	return tp, nil
}
