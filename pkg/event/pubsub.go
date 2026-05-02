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
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/stats"
	"github.com/flowline-io/flowbot/pkg/trace"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.uber.org/fx"
)

var logger = flog.WatermillLogger

func NewSubscriber(lc fx.Lifecycle) (message.Subscriber, error) {
	client, err := newRedisClient()
	if err != nil {
		return nil, err
	}
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
		OnStart: func(ctx context.Context) error {
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return subscriber.Close()
		},
	})

	return subscriber, err
}

var Publisher message.Publisher

func NewPublisher(lc fx.Lifecycle) (message.Publisher, error) {
	var err error
	client, err := newRedisClient()
	if err != nil {
		return nil, err
	}
	Publisher, err = redisstream.NewPublisher(
		redisstream.PublisherConfig{
			Client:     client,
			Marshaller: redisstream.DefaultMarshallerUnmarshaller{},
		},
		logger,
	)

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return Publisher.Close()
		},
	})

	return Publisher, err
}

func NewRouter(_ *redis.Client, _ *sdktrace.TracerProvider) (*message.Router, error) {
	router, err := message.NewRouter(message.RouterConfig{}, logger)
	if err != nil {
		return nil, err
	}

	router.AddMiddleware(
		middleware.CorrelationID,
		middleware.Timeout(10*time.Minute),
		Retry{
			MaxRetries:          3,
			InitialInterval:     1 * time.Second,
			MaxInterval:         30 * time.Second,
			Multiplier:          2.0,
			MaxElapsedTime:      2 * time.Minute,
			RandomizationFactor: 0.5,
			OnRetryHook: func(retryNum int, delay time.Duration) {
				flog.Info("Retry attempt #%d, waiting %v before next retry", retryNum, delay)
			},
			Logger: logger,
		}.Middleware,
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

func NewMessage(payload any) (*message.Message, error) {
	data, err := sonic.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	msg := message.NewMessage(watermill.NewUUID(), data)
	middleware.SetCorrelationID(watermill.NewShortUUID(), msg)

	return msg, nil
}

func PublishMessage(ctx context.Context, topic string, payload any) error {
	msg, err := NewMessage(payload)
	if err != nil {
		return fmt.Errorf("failed to new message: %w", err)
	}

	_, publishSpan := trace.StartSpan(ctx, "event.publish "+topic,
		attribute.String("messaging.destination", topic),
		attribute.String("messaging.message.id", msg.UUID),
	)
	defer publishSpan.End()

	carrier := propagation.MapCarrier{}
	otel.GetTextMapPropagator().Inject(ctx, carrier)
	for k, v := range carrier {
		msg.Metadata.Set(k, v)
	}
	msg.Metadata.Set("x-otel-topic", topic)

	err = Publisher.Publish(topic, msg)
	if err != nil {
		publishSpan.RecordError(err)
		publishSpan.SetStatus(codes.Error, err.Error())
	}
	return err
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
