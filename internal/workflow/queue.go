package workflow

import (
	"context"
	"fmt"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/hibiken/asynq"
	"runtime"
	"time"
)

const (
	TypeCron = "cron"
	TypeJob  = "job"

	cronQueueName = "workflow_cron"
	jobQueueName  = "workflow_job"
)

type Task struct {
	ID    string
	Queue string
	Task  *asynq.Task
}

func defaultRedisClientOpt() asynq.RedisClientOpt {
	return asynq.RedisClientOpt{
		Addr:     fmt.Sprintf("%s:%d", config.App.Redis.Host, config.App.Redis.Port),
		Password: config.App.Redis.Password,
	}
}

func PushTask(t *Task) error {
	client := asynq.NewClient(defaultRedisClientOpt())
	info, err := client.Enqueue(t.Task,
		asynq.Queue(t.Queue),
		asynq.TaskID(t.ID),
		asynq.MaxRetry(3),
		asynq.Retention(3*24*time.Hour),
	)
	if err != nil {
		return err
	}
	flog.Info("Enqueued %s, ID:%s", t.Task.Type(), info.ID)
	return nil
}

type Queue struct {
	srv *asynq.Server
}

func NewQueue() *Queue {
	srv := asynq.NewServer(defaultRedisClientOpt(), asynq.Config{
		Logger:      flog.AsynqLogger,
		LogLevel:    flog.AsynqLogLevel(config.App.Log.Level),
		Concurrency: runtime.NumCPU() * 2,
		Queues: map[string]int{
			jobQueueName: 10,
		},
	})
	return &Queue{srv: srv}
}

func (q *Queue) Run() {
	mux := asynq.NewServeMux()
	mux.Use(loggingMiddleware)
	mux.HandleFunc(TypeCron, HandleCronTask)
	mux.HandleFunc(TypeJob, HandleJobTask)

	if err := q.srv.Start(mux); err != nil {
		flog.Fatal("task queue failed %v", err)
	}
}

func (q *Queue) Shutdown() {
	q.srv.Shutdown()
}

func loggingMiddleware(h asynq.Handler) asynq.Handler {
	return asynq.HandlerFunc(func(ctx context.Context, t *asynq.Task) error {
		start := time.Now()
		flog.Debug("Start processing %q", t.Type())
		err := h.ProcessTask(ctx, t)
		if err != nil {
			flog.Error(fmt.Errorf("failed processing %q: Elapsed Time = %v, Payload = %s, Error = %v",
				t.Type(), time.Since(start), string(t.Payload()), err))
			return err
		}
		flog.Debug("finished processing %q: Elapsed Time = %v, Payload = %s",
			t.Type(), time.Since(start), string(t.Payload()))
		return nil
	})
}
