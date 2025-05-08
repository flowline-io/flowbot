package event

import (
	"context"
	"fmt"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-redisstream/pkg/redisstream"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/ThreeDotsLabs/watermill/message/router/plugin"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/rdb"
	"github.com/flowline-io/flowbot/pkg/stats"
	jsoniter "github.com/json-iterator/go"
)

var logger = flog.WatermillLogger

func NewSubscriber() (message.Subscriber, error) {
	return redisstream.NewSubscriber(
		redisstream.SubscriberConfig{
			Client:       rdb.Client,
			Unmarshaller: redisstream.DefaultMarshallerUnmarshaller{},
		},
		logger,
	)
}

func NewPublisher() (message.Publisher, error) {
	return redisstream.NewPublisher(
		redisstream.PublisherConfig{
			Client:     rdb.Client,
			Marshaller: redisstream.DefaultMarshallerUnmarshaller{},
		},
		logger,
	)
}

func NewRouter() (*message.Router, error) {
	router, err := message.NewRouter(message.RouterConfig{}, logger)
	if err != nil {
		return nil, err
	}

	router.AddPlugin(plugin.SignalsHandler)

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

	router.AddMiddleware(func(h message.HandlerFunc) message.HandlerFunc {
		return func(message *message.Message) ([]*message.Message, error) {
			flog.Debug("executing handler specific middleware for %s", message.UUID)
			// metrics
			stats.EventTotalCounter().Inc()
			// handle
			return h(message)
		}
	})

	return router, nil
}

func NewMessage(payload any) (*message.Message, error) {
	data, err := jsoniter.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	msg := message.NewMessage(watermill.NewUUID(), data)
	middleware.SetCorrelationID(watermill.NewShortUUID(), msg) // todo option with value

	return msg, nil
}

func PublishMessage(ctx context.Context, topic string, payload any) error {
	msg, err := NewMessage(payload)
	if err != nil {
		return fmt.Errorf("failed to new message: %w", err)
	}

	publisher, err := NewPublisher()
	if err != nil {
		return fmt.Errorf("failed to new publisher: %w", err)
	}

	return publisher.Publish(topic, msg)
}
