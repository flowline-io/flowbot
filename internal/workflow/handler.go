package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/hibiken/asynq"
	"strconv"
)

func NewJobTask(job *model.Job) (*Task, error) {
	payload, err := json.Marshal(types.JobInfo{
		Job: job,
	})
	if err != nil {
		return nil, err
	}
	return &Task{
		ID:    strconv.FormatInt(job.ID, 10),
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
		ID:    strconv.FormatInt(step.ID, 10),
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
		ID:    strconv.FormatInt(step.ID, 10),
		Queue: workerQueueName,
		Task:  asynq.NewTask(TypeWorker, payload),
	}, nil
}

func HandleCronTask(_ context.Context, t *asynq.Task) error {
	var trigger model.WorkflowTrigger
	if err := json.Unmarshal(t.Payload(), &trigger); err != nil {
		return fmt.Errorf("failed to unmarshal trigger: %v: %w", err, asynq.SkipRetry)
	}
	flog.Info("trigger: %v", trigger)
	flog.Info("%s task has been received", t.Type())

	// get workflow
	workflow, err := store.Chatbot.GetWorkflow(trigger.WorkflowID)
	if err != nil {
		flog.Error(err)
		return err
	}
	if workflow.State == model.WorkflowDisable {
		flog.Debug("workflow %d is disabled", workflow.ID)
		return nil
	}
	// create job
	dagId := int64(0)
	if len(workflow.Dag) > 0 {
		dagId = workflow.Dag[0].ID
	}
	_, err = store.Chatbot.CreateJob(&model.Job{
		UID:        workflow.UID,
		Topic:      workflow.Topic,
		WorkflowID: workflow.ID,
		DagID:      dagId,
		TriggerID:  trigger.ID,
		State:      model.JobReady,
	})

	return err
}

func HandleJobTask(_ context.Context, t *asynq.Task) error {
	var job types.JobInfo
	if err := json.Unmarshal(t.Payload(), &job); err != nil {
		return fmt.Errorf("failed to unmarshal job: %v: %w", err, asynq.SkipRetry)
	}
	flog.Info("job: %v", job)
	flog.Info("%s task has been received", t.Type())

	job.FSM = NewJobFSM(job.Job.State)
	return job.FSM.Event(context.Background(), "run", job.Job)
}

func HandleStepTask(_ context.Context, t *asynq.Task) error {
	var step types.StepInfo
	if err := json.Unmarshal(t.Payload(), &step); err != nil {
		return fmt.Errorf("failed to unmarshal job: %v: %w", err, asynq.SkipRetry)
	}
	flog.Info("step: %v", step)
	flog.Info("%s task has been received", t.Type())

	return nil
}

func HandleWorkerTask(_ context.Context, t *asynq.Task) error {
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
		_, _ = t.ResultWriter().Write([]byte("success"))
		return step.FSM.Event(context.Background(), "success", step.Step)
	}
}
