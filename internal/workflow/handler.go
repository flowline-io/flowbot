package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/hibiken/asynq"
)

func NewJobTask(job *model.Job) (*Task, error) {
	payload, err := json.Marshal(types.JobInfo{
		Job: job,
	})
	if err != nil {
		return nil, err
	}
	return &Task{
		ID:    job.UID,
		Queue: jobQueueName,
		Task:  asynq.NewTask(TypeJob, payload),
	}, nil
}

func NewStepTask(step *model.Step) (*Task, error) {
	payload, err := json.Marshal(types.StepInfo{
		Step: step,
	})
	if err != nil {
		return nil, err
	}
	return &Task{
		ID:    step.UID,
		Queue: stepQueueName,
		Task:  asynq.NewTask(TypeStep, payload),
	}, nil
}

func NewWorkerTask(step *model.Step) (*Task, error) {
	payload, err := json.Marshal(types.StepInfo{
		Step: step,
	})
	if err != nil {
		return nil, err
	}
	return &Task{
		ID:    step.UID,
		Queue: workerQueueName,
		Task:  asynq.NewTask(TypeWorker, payload),
	}, nil
}

func HandleJobTask(ctx context.Context, t *asynq.Task) error {
	var job types.JobInfo
	if err := json.Unmarshal(t.Payload(), &job); err != nil {
		return fmt.Errorf("failed to unmarshal job: %v: %w", err, asynq.SkipRetry)
	}
	flog.Info("job: %v", job)

	flog.Info("%s task has been received", t.Type())

	job.FSM = NewJobFSM(job.Job.State)
	return job.FSM.Event(context.Background(), "run", job.Job)
}

func HandleStepTask(ctx context.Context, t *asynq.Task) error {
	var step types.StepInfo
	if err := json.Unmarshal(t.Payload(), &step); err != nil {
		return fmt.Errorf("failed to unmarshal job: %v: %w", err, asynq.SkipRetry)
	}
	flog.Info("step: %v", step)

	flog.Info("%s task has been received", t.Type())

	return nil
}

func HandleWorkerTask(ctx context.Context, t *asynq.Task) error {
	var step types.StepInfo
	if err := json.Unmarshal(t.Payload(), &step); err != nil {
		return fmt.Errorf("failed to unmarshal job: %v: %w", err, asynq.SkipRetry)
	}
	flog.Info("step: %v", step)

	flog.Info("%s task has been received", t.Type())

	step.FSM = NewStepFSM(step.Step.State)
	err := step.FSM.Event(context.Background(), "run", step.Step)
	if err != nil {
		flog.Error(err)
		return step.FSM.Event(context.Background(), "error", step.Step)
	} else {
		return step.FSM.Event(context.Background(), "success", step.Step)
	}
}
