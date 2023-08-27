package queue

import (
	"encoding/json"
	"os"
	"time"

	"github.com/adjust/rmq/v5"
	"github.com/redis/go-redis/v9"
	"github.com/sysatom/flowbot/internal/types"
	"github.com/sysatom/flowbot/pkg/logs"
)

const (
	prefetchLimit = 1000
	pollDuration  = 100 * time.Millisecond
)

var connection rmq.Connection

func init() {
	addr := os.Getenv("REDIS_ADDR")
	password := os.Getenv("REDIS_PASSWORD")
	if addr == "" || password == "" {
		panic("redis config error")
	}

	errChan := make(chan error, 10)
	go logErrors(errChan)

	client := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     password,
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
	logs.Info.Println("message queue stopped")
}

func logErrors(errChan <-chan error) {
	for err := range errChan {
		switch err := err.(type) {
		case *rmq.HeartbeatError:
			if err.Count == rmq.HeartbeatErrorLimit {
				logs.Err.Println("heartbeat error (limit): ", err)
			} else {
				logs.Err.Println("heartbeat error: ", err)
			}
		case *rmq.ConsumeError:
			logs.Err.Println("consume error: ", err)
		case *rmq.DeliveryError:
			logs.Err.Println("delivery error: ", err.Delivery, err)
		default:
			logs.Err.Println("other error: ", err)
		}
	}
}

func AsyncMessage(rcptTo, original string, msg types.MsgPayload) error {
	botUid := types.ParseUserId(original)
	qp, err := types.ConvertQueuePayload(rcptTo, botUid.UserId(), msg)
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
