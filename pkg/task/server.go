package task

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/hibiken/asynq"
	"strconv"
)

const TypeExample = "example"
const TypeDemo = "demo"

type Task struct {
	ID    string
	Queue string
	Task  *asynq.Task
}

func HandleTask(ctx context.Context, t *asynq.Task) error {

	flog.Info("%s task has been received", t.Type())

	switch t.Type() {
	case TypeExample:
		var job model.Job
		if err := json.Unmarshal(t.Payload(), &job); err != nil {
			return fmt.Errorf("failed to unmarshal job: %v: %w", err, asynq.SkipRetry)
		}
		flog.Info("job: %v", job)
	case TypeDemo:
		var step model.Step
		if err := json.Unmarshal(t.Payload(), &step); err != nil {
			return fmt.Errorf("failed to unmarshal job: %v: %w", err, asynq.SkipRetry)
		}
		flog.Info("step: %v", step)
		return errors.New("demo error")
	}

	return nil
}

func NewExampleTask(id int64) (*Task, error) { // todo
	payload, err := json.Marshal(model.Job{
		ID: id,
	})
	if err != nil {
		return nil, err
	}
	return &Task{
		ID:    strconv.FormatInt(id, 10),
		Queue: jobQueueName,
		Task:  asynq.NewTask(TypeExample, payload),
	}, nil
}
func NewDemoTask(id int64) (*Task, error) { // todo
	payload, err := json.Marshal(model.Step{
		ID: id,
	})
	if err != nil {
		return nil, err
	}
	return &Task{
		ID:    strconv.FormatInt(id, 10),
		Queue: stepQueueName,
		Task:  asynq.NewTask(TypeDemo, payload),
	}, nil
}

const jobQueueName = "workflow-job"
const stepQueueName = "workflow-step"

func PushTask(t *Task) error {
	client := asynq.NewClient(asynq.RedisClientOpt{Addr: fmt.Sprintf("%s:%d", config.App.Redis.Host, config.App.Redis.Port), Password: config.App.Redis.Password}) // todo client struct
	info, err := client.Enqueue(t.Task,
		asynq.Queue(t.Queue),
		asynq.TaskID(t.ID),
		asynq.MaxRetry(3),
	) // todo options
	if err != nil {
		return err
	}
	flog.Info("Enqueued %s: %d", t.Task.Type(), info.ID)
	return nil
}

func Init() {
	srv := asynq.NewServer(asynq.RedisClientOpt{Addr: fmt.Sprintf("%s:%d", config.App.Redis.Host, config.App.Redis.Port), Password: config.App.Redis.Password}, asynq.Config{
		Concurrency: 10,
		Queues: map[string]int{
			jobQueueName:  6,
			stepQueueName: 4,
		},
	})
	go func() {
		err := srv.Run(asynq.HandlerFunc(HandleTask))
		if err != nil {
			flog.Fatal("task queue failed %v", err)
		}
	}()
}
