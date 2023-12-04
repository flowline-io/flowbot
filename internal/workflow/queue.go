package workflow

import (
	"context"
	"fmt"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/hibiken/asynq"
	"time"
)

const (
	TypeCron   = "cron"
	TypeJob    = "job"
	TypeStep   = "step"
	TypeWorker = "worker"

	cronQueueName   = "workflow_cron"
	jobQueueName    = "workflow_job"
	stepQueueName   = "workflow_step"
	workerQueueName = "workflow_worker"

	jobPriority  = 6
	stepPriority = 4
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
	) // todo options
	if err != nil {
		return err
	}
	flog.Info("Enqueued %s: %s", t.Task.Type(), info.ID)
	return nil
}

type Queue struct {
	srv *asynq.Server
}

func NewQueue() *Queue {
	srv := asynq.NewServer(defaultRedisClientOpt(), asynq.Config{
		Concurrency: 2,
		Queues: map[string]int{
			jobQueueName:  jobPriority,
			stepQueueName: stepPriority,
		},
		// todo options
	})
	return &Queue{srv: srv}
}

func (q *Queue) Run() {
	mux := asynq.NewServeMux()
	mux.Use(loggingMiddleware)
	mux.HandleFunc(TypeCron, HandleCronTask)
	mux.HandleFunc(TypeJob, HandleJobTask)
	mux.HandleFunc(TypeStep, HandleStepTask)
	mux.HandleFunc(TypeWorker, HandleWorkerTask)

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
		flog.Info("Start processing %q", t.Type())
		err := h.ProcessTask(ctx, t)
		if err != nil {
			return err
		}
		flog.Info("Finished processing %q: Elapsed Time = %v", t.Type(), time.Since(start))
		return nil
	})
}