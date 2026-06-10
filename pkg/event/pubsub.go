// Package event provides Watermill-based publish/subscribe infrastructure backed by Redis Streams.
package event

import (
	"context"
	"fmt"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-redisstream/pkg/redisstream"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/bytedance/sonic"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.uber.org/fx"

	"github.com/flowline-io/flowbot/pkg/backoff"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/stats"
)

var logger = flog.WatermillLogger

// NewSubscriber creates a Watermill Redis Stream subscriber using the shared Redis client.
func NewSubscriber(lc fx.Lifecycle, client *redis.Client) (message.Subscriber, error) {
	subscriber, err := redisstream.NewSubscriber(
		redisstream.SubscriberConfig{
			Client:       client,
			Unmarshaller: redisstream.DefaultMarshallerUnmarshaller{},
		},
		logger,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create redis subscriber: %w", err)
	}

	lc.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			return nil
		},
		OnStop: func(_ context.Context) error {
			return subscriber.Close()
		},
	})

	return subscriber, err
}

// Publisher is the global Watermill publisher, provided by NewPublisher via fx.
var Publisher message.Publisher

// NewPublisher creates a Watermill Redis Stream publisher using the shared Redis client.
func NewPublisher(lc fx.Lifecycle, client *redis.Client) (message.Publisher, error) {
	pub, err := redisstream.NewPublisher(
		redisstream.PublisherConfig{
			Client:     client,
			Marshaller: redisstream.DefaultMarshallerUnmarshaller{},
		},
		logger,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create redis publisher: %w", err)
	}

	Publisher = pub

	lc.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			return nil
		},
		OnStop: func(_ context.Context) error {
			return pub.Close()
		},
	})

	return pub, nil
}

// NewRouter creates a Watermill message router with standard middleware.
func NewRouter(_ *sdktrace.TracerProvider) (*message.Router, error) {
	router, err := message.NewRouter(message.RouterConfig{}, logger)
	if err != nil {
		return nil, err
	}

	router.AddMiddleware(
		middleware.CorrelationID,
		middleware.Timeout(10*time.Minute),
		backoff.Middleware(backoff.Config{
			MaxAttempts:     4, // 1 initial + 3 retries
			InitialInterval: 1 * time.Second,
			MaxInterval:     30 * time.Second,
			Multiplier:      2.0,
			MaxElapsedTime:  2 * time.Minute,
			Jitter:          true,
			OnRetry: func(attempt int, delay time.Duration, _ error) {
				flog.Info("Retry attempt #%d, waiting %v before next retry", attempt, delay)
			},
		}, logger),
		middleware.Recoverer,
	)

	router.AddMiddleware(TraceConsumerMiddleware())

	router.AddMiddleware(func(h message.HandlerFunc) message.HandlerFunc {
		return func(message *message.Message) ([]*message.Message, error) {
			flog.Debug("executing handler specific middleware for %s", message.UUID)
			stats.EventTotalCounter().Inc()
			return h(message)
		}
	})

	return router, nil
}

// NewMessage creates a Watermill message from the given payload, marshaled as JSON.
func NewMessage(payload any) (*message.Message, error) {
	data, err := sonic.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	msg := message.NewMessage(watermill.NewUUID(), data)
	middleware.SetCorrelationID(watermill.NewShortUUID(), msg)

	return msg, nil
}

// PublishMessage publishes a message to the given topic using the global Publisher, with OpenTelemetry tracing.
func PublishMessage(ctx context.Context, topic string, payload any) error {
	return publishWith(ctx, Publisher, topic, payload)
}

// TraceConsumerMiddleware returns a Watermill middleware that extracts OTel trace context
// from message metadata and creates a consumer span for each incoming message.
func TraceConsumerMiddleware() message.HandlerMiddleware {
	prop := otel.GetTextMapPropagator()
	tracer := otel.Tracer("watermill")

	return func(h message.HandlerFunc) message.HandlerFunc {
		return func(msg *message.Message) ([]*message.Message, error) {
			carrier := propagation.MapCarrier{}
			for k, v := range msg.Metadata {
				carrier.Set(k, v)
			}
			ctx := prop.Extract(msg.Context(), carrier)

			topic := ""
			if t := msg.Metadata.Get("x-otel-topic"); t != "" {
				topic = t
				delete(msg.Metadata, "x-otel-topic")
			}

			spanName := "event.receive"
			if topic != "" {
				spanName = "event.receive " + topic
			}

			ctx, span := tracer.Start(ctx, spanName)
			span.SetAttributes(
				attribute.String("messaging.operation", "receive"),
				attribute.String("messaging.message.id", msg.UUID),
			)
			if topic != "" {
				span.SetAttributes(attribute.String("messaging.destination", topic))
			}
			msg.SetContext(ctx)
			defer span.End()

			return h(msg)
		}
	}
}
