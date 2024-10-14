package event

import (
	"context"
	"log"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-redisstream/pkg/redisstream"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/ThreeDotsLabs/watermill/message/router/plugin"
	"github.com/flowline-io/flowbot/pkg/cache"
	"github.com/flowline-io/flowbot/pkg/flog"
	jsoniter "github.com/json-iterator/go"
)

var logger = flog.WatermillLogger

func NewSubscriber() (message.Subscriber, error) {
	return redisstream.NewSubscriber(
		redisstream.SubscriberConfig{
			Client:       cache.DB,
			Unmarshaller: redisstream.DefaultMarshallerUnmarshaller{},
		},
		logger,
	)
}

func NewPublisher() (message.Publisher, error) {
	return redisstream.NewPublisher(
		redisstream.PublisherConfig{
			Client:     cache.DB,
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
		middleware.Timeout(30*time.Second),
		middleware.Retry{
			MaxRetries:      3,
			InitialInterval: 100 * time.Millisecond,
			Logger:          logger,
		}.Middleware,
		middleware.Recoverer,
	)

	router.AddMiddleware(func(h message.HandlerFunc) message.HandlerFunc {
		return func(message *message.Message) ([]*message.Message, error) {
			log.Println("executing handler specific middleware for ", message.UUID)

			return h(message)
		}
	})

	return router, nil
}

func NewMessage(payload any) (*message.Message, error) {
	data, err := jsoniter.Marshal(payload)
	if err != nil {
		return nil, err
	}

	msg := message.NewMessage(watermill.NewUUID(), data)
	middleware.SetCorrelationID(watermill.NewShortUUID(), msg) // todo option with value

	return msg, nil
}

func PublishMessage(ctx context.Context, topic string, payload any) error {
	msg, err := NewMessage(payload)
	if err != nil {
		return err
	}

	publisher, err := NewPublisher()
	if err != nil {
		return err
	}
	if err := publisher.Publish(topic, msg); err != nil {
		return err
	}

	return nil
}
