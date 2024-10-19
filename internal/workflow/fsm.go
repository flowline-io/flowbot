package workflow

import (
	"context"
	"errors"
	"fmt"
	"github.com/flowline-io/flowbot/internal/bots"
	"github.com/flowline-io/flowbot/internal/ruleset/workflow"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/pkg/dag"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/looplab/fsm"
)

func NewJobFSM(state model.JobState) *fsm.FSM {
	initial := "created"
	switch state {
	case model.JobReady:
		initial = "ready"
	case model.JobStart:
		initial = "start"
	case model.JobRunning:
		initial = "running"
	case model.JobSucceeded:
		initial = "succeeded"
	case model.JobCanceled:
		initial = "canceled"
	case model.JobFailed:
		initial = "failed"
	case model.JobStateUnknown:
		initial = "unknown"
	}
	f := fsm.NewFSM(
		initial,
		fsm.Events{
			{Name: "queue", Src: []string{"ready"}, Dst: "start"},
			{Name: "run", Src: []string{"start"}, Dst: "running"},
			{Name: "success", Src: []string{"running"}, Dst: "succeeded"},
			{Name: "cancel", Src: []string{"running"}, Dst: "canceled"},
			{Name: "error", Src: []string{"running"}, Dst: "failed"},
		},
		fsm.Callbacks{
			"before_run": func(_ context.Context, e *fsm.Event) {
				var job *model.Job
				for _, item := range e.Args {
					if m, ok := item.(*model.Job); ok {
						job = m
					}
				}
				if job == nil {
					e.Cancel(errors.New("error job"))
					return
				}

				err := store.Database.UpdateJobState(job.ID, model.JobRunning)
				if err != nil {
					e.Cancel(err)
					e.Err = fmt.Errorf("failed to update job state %d, %w", job.ID, err)
					return
				}

				// split dag
				d, err := store.Database.GetDag(job.DagID)
				if err != nil {
					e.Cancel(err)
					e.Err = err
					return
				}
				list, err := dag.TopologySort(d)
				if err != nil {
					e.Cancel(err)
					e.Err = fmt.Errorf("failed to topology sort dag %d, %w", job.DagID, err)
					return
				}

				// create steps
				steps := make([]*model.Step, 0, len(list))
				findSteps, err := store.Database.GetStepsByJobId(job.ID)
				if err != nil {
					e.Cancel(err)
					e.Err = err
					return
				}
				if len(findSteps) == 0 {
					for _, step := range list {
						steps = append(steps, &model.Step{
							UID:    job.UID,
							Topic:  job.Topic,
							JobID:  job.ID,
							Action: step.Action,
							Name:   step.Name,
							State:  model.StepReady,
							NodeID: step.NodeID,
							Depend: step.Depend,
						})
					}
					err = store.Database.CreateSteps(steps)
					if err != nil {
						e.Cancel(err)
						e.Err = fmt.Errorf("failed to create steps for job %d, %w", job.ID, err)
						return
					}
				} else {
					steps = findSteps
				}

				// running count
				err = store.Database.IncreaseWorkflowCount(job.WorkflowID, 0, 0, 1, 0)
				if err != nil {
					e.Cancel(err)
					e.Err = err
					return
				}

				// run step
				for _, step := range steps {
					// start
					step.State = model.StepStart
					err = store.Database.UpdateStep(step.ID, &model.Step{
						EndedAt: nil,
						State:   model.StepStart,
					})
					if err != nil {
						e.Cancel(err)
						e.Err = fmt.Errorf("failed to update step state %d, %w", step.ID, err)
						return
					}

					// depend
					dependSteps, err := store.Database.GetStepsByDepend(step.JobID, step.Depend)
					if err != nil {
						e.Cancel(err)
						e.Err = fmt.Errorf("failed to get depend steps for step %d, %w", step.ID, err)
						return
					}
					allFinished := true
					mergeOutput := types.KV{}
					for _, dependStep := range dependSteps {
						switch dependStep.State {
						case model.StepCreated, model.StepReady, model.StepStart, model.StepRunning:
							allFinished = false
						case model.StepSucceeded:
							// merge output
							mergeOutput = mergeOutput.Merge(types.KV(dependStep.Output))
						case model.StepFailed, model.StepCanceled, model.StepSkipped:
							err = store.Database.UpdateStepState(step.ID, dependStep.State)
							if err != nil {
								flog.Error(err)
							}
							allFinished = false
						default:
							allFinished = false
						}
					}
					if len(dependSteps) > 0 && allFinished {
						for _, dependStep := range dependSteps {
							flog.Debug("step %d depend steps: %v", step.ID, dependStep)
						}
						flog.Debug("all depend step finished for step %d output: %v", step.ID, mergeOutput)
						step.Input = model.JSON(mergeOutput)
						err = store.Database.UpdateStep(step.ID, &model.Step{
							Input: model.JSON(mergeOutput),
						})
						if err != nil {
							e.Cancel(err)
							e.Err = fmt.Errorf("failed to update step input %d, %w", step.ID, err)
							return
						}
					}

					// run
					stepFSM := NewStepFSM(step.State)
					err = stepFSM.Event(context.Background(), "run", step)
					if err != nil {
						_ = stepFSM.Event(context.Background(), "error", step, err)
					} else {
						err = stepFSM.Event(context.Background(), "success", step)
					}
					if err != nil {
						e.Cancel(err)
						e.Err = fmt.Errorf("failed to run step %d, %w", step.ID, err)
						return
					}
				}
			},
			"before_success": func(ctx context.Context, e *fsm.Event) {
				var job *model.Job
				for _, item := range e.Args {
					if m, ok := item.(*model.Job); ok {
						job = m
					}
				}
				if job == nil {
					e.Cancel(errors.New("error job"))
					return
				}

				err := store.Database.UpdateJobState(job.ID, model.JobSucceeded)
				if err != nil {
					e.Cancel(err)
					e.Err = err
					return
				}
				err = DeleteJob(ctx, job)
				if err != nil {
					e.Cancel(err)
					e.Err = fmt.Errorf("failed to delete job %d, %w", job.ID, err)
					return
				}
				// successful count
				err = store.Database.IncreaseWorkflowCount(job.WorkflowID, 1, 0, -1, 0)
				if err != nil {
					e.Cancel(err)
					e.Err = err
					return
				}
			},
			"before_error": func(_ context.Context, e *fsm.Event) {
				var job *model.Job
				var err error
				for _, item := range e.Args {
					switch v := item.(type) {
					case *model.Job:
						job = v
					case error:
						err = v
					default:
						e.Cancel(errors.New("error args type"))
						return
					}
				}
				if job == nil {
					e.Cancel(errors.New("error job"))
					return
				}
				if err == nil {
					e.Cancel(errors.New("error err"))
					return
				} else {
					flog.Error(err)
				}

				err = store.Database.UpdateJobState(job.ID, model.JobFailed)
				if err != nil {
					e.Cancel(err)
					e.Err = fmt.Errorf("failed to update job state %d, %w", job.ID, err)
					return
				}
				// failed count
				err = store.Database.IncreaseWorkflowCount(job.WorkflowID, 0, 1, -1, 0)
				if err != nil {
					e.Cancel(err)
					e.Err = fmt.Errorf("failed to increase workflow count %d, %w", job.WorkflowID, err)
					return
				}
			},
			"before_cancel": func(_ context.Context, e *fsm.Event) {
				var job *model.Job
				for _, item := range e.Args {
					if m, ok := item.(*model.Job); ok {
						job = m
					}
				}
				if job == nil {
					e.Cancel(errors.New("error job"))
					return
				}

				err := store.Database.UpdateJobState(job.ID, model.JobCanceled)
				if err != nil {
					e.Cancel(err)
					e.Err = fmt.Errorf("failed to update job state %d, %w", job.ID, err)
					return
				}
				// successful count
				err = store.Database.IncreaseWorkflowCount(job.WorkflowID, 0, 0, -1, 1)
				if err != nil {
					e.Cancel(err)
					e.Err = fmt.Errorf("failed to increase workflow count %d, %w", job.WorkflowID, err)
					return
				}
			},
		},
	)
	return f
}

func NewStepFSM(state model.StepState) *fsm.FSM {
	initial := "created"
	switch state {
	case model.StepCreated:
		initial = "created"
	case model.StepReady:
		initial = "ready"
	case model.StepStart:
		initial = "start"
	case model.StepRunning:
		initial = "running"
	case model.StepSucceeded:
		initial = "succeeded"
	case model.StepCanceled:
		initial = "canceled"
	case model.StepFailed:
		initial = "failed"
	case model.StepSkipped:
		initial = "skipped"
	case model.StepStateUnknown:
		initial = "unknown"
	}
	f := fsm.NewFSM(
		initial,
		fsm.Events{
			{Name: "bind", Src: []string{"created"}, Dst: "ready"},
			{Name: "queue", Src: []string{"ready"}, Dst: "start"},
			{Name: "run", Src: []string{"start"}, Dst: "running"},
			{Name: "success", Src: []string{"running"}, Dst: "succeeded"},
			{Name: "error", Src: []string{"running"}, Dst: "failed"},
			{Name: "cancel", Src: []string{"running"}, Dst: "canceled"},
			{Name: "skip", Src: []string{"running"}, Dst: "skipped"},
		},
		fsm.Callbacks{
			"before_run": func(_ context.Context, e *fsm.Event) {
				var step *model.Step
				for _, item := range e.Args {
					if m, ok := item.(*model.Step); ok {
						step = m
					}
				}
				if step == nil {
					e.Cancel(errors.New("error step"))
					return
				}

				err := store.Database.UpdateStepState(step.ID, model.StepRunning)
				if err != nil {
					e.Cancel(err)
					e.Err = err
					return
				}

				// run step
				bot, _ := types.KV(step.Action).String("bot")
				ruleId, _ := types.KV(step.Action).String("rule_id")

				var botHandler bots.Handler
				for name, handler := range bots.List() {
					if bot != name {
						continue
					}
					for _, item := range handler.Rules() {
						switch v := item.(type) {
						case []workflow.Rule:
							for _, rule := range v {
								if rule.Id == ruleId {
									botHandler = handler
								}
							}
						}
					}
				}
				if botHandler == nil {
					e.Err = errors.New("bot handler not found")
					return
				}
				ctx := types.Context{
					AsUser:         types.Uid(step.UID),
					Topic:          step.Topic,
					WorkflowRuleId: ruleId,
				}
				parameters, _ := types.KV(step.Action).Any("parameters")
				if p, ok := parameters.(map[string]interface{}); ok {
					if step.Input == nil {
						step.Input = p
					} else {
						step.Input = model.JSON(types.KV(step.Input).Merge(p))
					}
				}
				output, err := botHandler.Workflow(ctx, types.KV(step.Input))
				if err != nil {
					e.Err = err
					return
				}

				// update output
				err = store.Database.UpdateStepOutput(step.ID, output)
				if err != nil {
					flog.Error(err)
				}
			},
			"before_success": func(_ context.Context, e *fsm.Event) {
				var step *model.Step
				for _, item := range e.Args {
					if m, ok := item.(*model.Step); ok {
						step = m
					}
				}
				if step == nil {
					e.Cancel(errors.New("error step"))
					return
				}

				err := store.Database.UpdateStepState(step.ID, model.StepSucceeded)
				if err != nil {
					e.Cancel(err)
					e.Err = err
					return
				}
			},
			"before_error": func(_ context.Context, e *fsm.Event) {
				var step *model.Step
				var err error
				for _, item := range e.Args {
					switch v := item.(type) {
					case *model.Step:
						step = v
					case error:
						err = v
					default:
						e.Cancel(errors.New("error args type"))
						return
					}
				}
				if step == nil {
					e.Cancel(errors.New("error step"))
					return
				}
				if err == nil {
					e.Cancel(errors.New("error err"))
					return
				}

				err = store.Database.UpdateStep(step.ID, &model.Step{
					State: model.StepFailed,
					Error: err.Error(),
				})
				if err != nil {
					e.Cancel(err)
					e.Err = err
					return
				}
			},
		},
	)

	return f
}
