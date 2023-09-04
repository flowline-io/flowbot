package queue

import (
	"encoding/json"
	"fmt"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/flog"
	"time"

	"github.com/adjust/rmq/v5"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/redis/go-redis/v9"
)

const (
	prefetchLimit = 1000
	pollDuration  = 100 * time.Millisecond
)

var connection rmq.Connection

func Init() {
	addr := fmt.Sprintf("%s:%d", config.App.Redis.Host, config.App.Redis.Port)
	password := config.App.Redis.Password
	if addr == "" || password == "" {
		panic("redis config error")
	}

	errChan := make(chan error, 10)
	go logErrors(errChan)

	client := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     password,
		DB:           config.App.Redis.DB,
		ReadTimeout:  time.Minute,
		WriteTimeout: time.Minute,
	})
	var err error
	connection, err = rmq.OpenConnectionWithRedisClient("consumer", client, errChan)
	if err != nil {
		panic(err)
	}
}

var messageQueue rmq.Queue

func InitMessageQueue(consumer rmq.Consumer) {
	var err error
	messageQueue, err = connection.OpenQueue("messages")
	if err != nil {
		panic(err)
	}

	if err = messageQueue.StartConsuming(prefetchLimit, pollDuration); err != nil {
		panic(err)
	}

	if _, err = messageQueue.AddConsumer("message", consumer); err != nil {
		panic(err)
	}
}

func Shutdown() {
	<-messageQueue.StopConsuming()
	flog.Info("message queue stopped")
}

func logErrors(errChan <-chan error) {
	for err := range errChan {
		switch err := err.(type) {
		case *rmq.HeartbeatError:
			if err.Count == rmq.HeartbeatErrorLimit {
				flog.Error(err)
			} else {
				flog.Error(err)
			}
		case *rmq.ConsumeError:
			flog.Error(err)
		case *rmq.DeliveryError:
			flog.Error(err)
		default:
			flog.Error(err)
		}
	}
}

func AsyncMessage(uid, topic string, msg types.MsgPayload) error {
	qp, err := types.ConvertQueuePayload(topic, uid, msg)
	if err != nil {
		return nil
	}
	payload, err := json.Marshal(qp)
	if err != nil {
		return nil
	}
	return messageQueue.PublishBytes(payload)
}

func Stats() (string, error) {
	queues, err := connection.GetOpenQueues()
	if err != nil {
		return "", err
	}

	stats, err := connection.CollectStats(queues)
	if err != nil {
		return "", err
	}

	return stats.GetHtml("", ""), nil
}
