package workflow

import (
	"context"
	"fmt"
	"strconv"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/hibiken/asynq"
	jsoniter "github.com/json-iterator/go"
)

func HandleCronTask(ctx context.Context, t *asynq.Task) error {
	var trigger model.WorkflowTrigger
	if err := jsoniter.Unmarshal(t.Payload(), &trigger); err != nil {
		return fmt.Errorf("failed to unmarshal trigger: %w: %w", err, asynq.SkipRetry)
	}
	flog.Debug("trigger %+v, %s task has been received", trigger, t.Type())

	// get workflow
	workflow, err := store.Database.GetWorkflow(trigger.WorkflowID)
	if err != nil {
		flog.Error(err)
		return fmt.Errorf("failed to get workflow %d, %w", trigger.WorkflowID, err)
	}
	if workflow.State == model.WorkflowDisable {
		flog.Debug("workflow %d is disabled", workflow.ID)
		return nil
	}

	// update trigger count
	err = store.Database.IncreaseWorkflowTriggerCount(trigger.ID, 1)
	if err != nil {
		flog.Error(err)
	}

	// create job
	dagId := int64(0)
	scriptVersion := int32(0)
	if len(workflow.Dag) > 0 {
		dagId = workflow.Dag[0].ID
		scriptVersion = workflow.Dag[0].ScriptVersion
	}
	job := &model.Job{
		UID:           workflow.UID,
		Topic:         workflow.Topic,
		WorkflowID:    workflow.ID,
		DagID:         dagId,
		TriggerID:     trigger.ID,
		ScriptVersion: scriptVersion,
		State:         model.JobReady,
	}
	_, err = store.Database.CreateJob(job)
	if err != nil {
		flog.Error(err)
		return fmt.Errorf("failed to create job %d, %w", job.ID, err)
	}
	err = SyncJob(ctx, job)
	if err != nil {
		flog.Error(err)
		return fmt.Errorf("failed to sync job %d, %w", job.ID, err)
	}

	return err
}

func NewJobTask(job *model.Job) (*Task, error) {
	payload, err := jsoniter.Marshal(job)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal job %d, %w", job.ID, err)
	}
	return &Task{
		ID:    strconv.FormatInt(job.ID, 10),
		Queue: jobQueueName,
		Task: asynq.NewTask(TypeJob, payload,
			asynq.MaxRetry(defaultMaxRetry),
			asynq.Retention(defaultRetention),
		),
	}, nil
}

func HandleJobTask(ctx context.Context, t *asynq.Task) error {
	var job *model.Job
	if err := jsoniter.Unmarshal(t.Payload(), &job); err != nil {
		return fmt.Errorf("failed to unmarshal job: %w: %w", err, asynq.SkipRetry)
	}
	flog.Debug("job: %+v", job)
	flog.Debug("%s task has been received", t.Type())

	fsm := NewJobFSM(job.State)
	err := fsm.Event(ctx, "run", job)
	if err != nil {
		fsm.SetState("running")
		return fsm.Event(ctx, "error", job, err)
	} else {
		return fsm.Event(ctx, "success", job)
	}
}
