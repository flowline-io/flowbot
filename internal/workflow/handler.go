package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/hibiken/asynq"
	"strconv"
)

func HandleCronTask(_ context.Context, t *asynq.Task) error {
	var trigger model.WorkflowTrigger
	if err := json.Unmarshal(t.Payload(), &trigger); err != nil {
		return fmt.Errorf("failed to unmarshal trigger: %v: %w", err, asynq.SkipRetry)
	}
	flog.Debug("trigger: %+v", trigger)
	flog.Debug("%s task has been received", t.Type())

	// get workflow
	workflow, err := store.Database.GetWorkflow(trigger.WorkflowID)
	if err != nil {
		flog.Error(err)
		return err
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
	if len(workflow.Dag) > 0 {
		dagId = workflow.Dag[0].ID
	}
	job := &model.Job{
		UID:        workflow.UID,
		Topic:      workflow.Topic,
		WorkflowID: workflow.ID,
		DagID:      dagId,
		TriggerID:  trigger.ID,
		State:      model.JobReady,
	}
	_, err = store.Database.CreateJob(job)
	if err != nil {
		flog.Error(err)
		return err
	}
	err = SyncJob(context.Background(), job)
	if err != nil {
		flog.Error(err)
		return err
	}

	return err
}

func NewJobTask(job *model.Job) (*Task, error) {
	payload, err := json.Marshal(job)
	if err != nil {
		return nil, err
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
	if err := json.Unmarshal(t.Payload(), &job); err != nil {
		return fmt.Errorf("failed to unmarshal job: %v: %w", err, asynq.SkipRetry)
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
