package workflow

import (
	"context"
	"errors"
	"github.com/flowline-io/flowbot/internal/bots"
	"github.com/flowline-io/flowbot/internal/ruleset/workflow"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/pkg/dag"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/looplab/fsm"
	"time"
)

func NewJobFSM(state model.JobState) *fsm.FSM {
	initial := "created"
	switch state {
	case model.JobReady:
		initial = "ready"
	case model.JobStart:
		initial = "start"
	case model.JobFinished:
		initial = "finished"
	case model.JobCanceled:
		initial = "canceled"
	case model.JobFailed:
		initial = "failed"
	}
	f := fsm.NewFSM(
		initial,
		fsm.Events{
			{Name: "run", Src: []string{"ready"}, Dst: "start"},
			{Name: "success", Src: []string{"start"}, Dst: "finished"},
			{Name: "cancel", Src: []string{"start"}, Dst: "canceled"},
			{Name: "error", Src: []string{"start"}, Dst: "failed"},
		},
		fsm.Callbacks{
			// split dag
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

				d, err := store.Chatbot.GetDag(job.DagID)
				if err != nil {
					e.Cancel(err)
					return
				}
				list, err := dag.TopologySort(d)
				if err != nil {
					e.Cancel(err)
					return
				}

				// create steps
				steps := make([]*model.Step, 0, len(list))
				for _, step := range list {
					m := &model.Step{
						UID:    job.UID,
						Topic:  job.Topic,
						JobID:  job.ID,
						Action: step.Action,
						Name:   step.Name,
						State:  step.State,
						NodeID: step.NodeID,
						Depend: step.Depend,
					}
					// update started at
					if step.State == model.StepReady {
						now := time.Now()
						m.StartedAt = &now
					}
					steps = append(steps, m)
				}
				err = store.Chatbot.CreateSteps(steps)
				if err != nil {
					e.Cancel(err)
					return
				}

				// update job state
				err = store.Chatbot.UpdateJobState(job.ID, model.JobStart)
				if err != nil {
					e.Cancel(err)
					return
				}
				// update job started at
				err = store.Chatbot.UpdateJobStartedAt(job.ID, time.Now())
				if err != nil {
					flog.Error(err)
				}
				// running count
				err = store.Chatbot.IncreaseWorkflowCount(job.WorkflowID, 0, 0, 1, 0)
				if err != nil {
					flog.Error(err)
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
	case model.StepRunning:
		initial = "running"
	case model.StepFinished:
		initial = "finished"
	case model.StepCanceled:
		initial = "canceled"
	case model.StepFailed:
		initial = "failed"
	case model.StepSkipped:
		initial = "skipped"
	}
	f := fsm.NewFSM(
		initial,
		fsm.Events{
			{Name: "bind", Src: []string{"created"}, Dst: "ready"},
			{Name: "run", Src: []string{"ready"}, Dst: "running"},
			{Name: "success", Src: []string{"running"}, Dst: "finished"},
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

				err := store.Chatbot.UpdateStepState(step.ID, model.StepRunning)
				if err != nil {
					e.Cancel(err)
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
					Original:       step.UID,
					RcptTo:         step.Topic,
					WorkflowRuleId: ruleId,
				}
				output, err := botHandler.Workflow(ctx, types.KV(step.Input))
				if err != nil {
					e.Err = err
					return
				}

				// update output
				err = store.Chatbot.UpdateStepOutput(int64(step.ID), output)
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

				err := store.Chatbot.UpdateStepState(step.ID, model.StepFinished)
				if err != nil {
					e.Cancel(err)
					return
				}
				// update finished at
				err = store.Chatbot.UpdateStepFinishedAt(step.ID, time.Now())
				if err != nil {
					e.Cancel(err)
					return
				}
			},
			"before_error": func(_ context.Context, e *fsm.Event) {
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

				err := store.Chatbot.UpdateStepState(step.ID, model.StepFailed)
				if err != nil {
					e.Cancel(err)
					return
				}
			},
		},
	)

	return f
}
