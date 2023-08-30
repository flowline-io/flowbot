package schedule

import (
	"context"
	"errors"
	"fmt"
	"github.com/flowline-io/flowbot/internal/bots"
	"github.com/flowline-io/flowbot/internal/ruleset/workflow"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/utils/parallelizer"
	"github.com/flowline-io/flowbot/pkg/utils/queue"
	"github.com/looplab/fsm"
	"time"
)

type Scheduler struct {
	stop chan struct{}

	SchedulingQueue *queue.DeltaFIFO

	//nextStartWorkerIndex int
}

func NewScheduler(queue *queue.DeltaFIFO) *Scheduler {
	s := &Scheduler{
		SchedulingQueue: queue,
		stop:            make(chan struct{}),
	}
	return s
}

func (sched *Scheduler) Run() {
	// ready step
	go parallelizer.JitterUntil(sched.pushReadyStep, time.Second, 0.0, true, sched.stop)
	// depend step
	go parallelizer.JitterUntil(sched.dependStep, 2*time.Second, 0.0, true, sched.stop)

	<-sched.stop
	flog.Info("scheduler stopped")
	sched.SchedulingQueue.Close()
}

func (sched *Scheduler) Shutdown() {
	sched.stop <- struct{}{}
}

func (sched *Scheduler) pushReadyStep() {
	list, err := store.Chatbot.GetStepsByState(model.StepReady)
	if err != nil {
		flog.Error(err)
		return
	}
	for _, step := range list {
		_, exists, err := sched.SchedulingQueue.GetByKey(stepKey(step))
		if err != nil {
			flog.Error(err)
			continue
		}
		if exists {
			continue
		}

		err = sched.SchedulingQueue.Add(&types.StepInfo{
			Step: step,
			FSM:  NewStepFSM(step.State),
		})
		if err != nil {
			flog.Error(err)
		}
	}
}

func (sched *Scheduler) dependStep() {
	list, err := store.Chatbot.GetStepsByState(model.StepCreated)
	if err != nil {
		flog.Error(err)
		return
	}
	for _, step := range list {
		dependSteps, err := store.Chatbot.GetStepsByDepend(int64(step.JobID), step.Depend)
		if err != nil {
			flog.Error(err)
			continue
		}
		allFinished := true
		mergeOutput := types.KV{}
		for _, dependStep := range dependSteps {
			switch dependStep.State {
			case model.StepCreated, model.StepReady, model.StepRunning:
				allFinished = false
			case model.StepFinished:
				// merge output
				mergeOutput = mergeOutput.Merge(types.KV(dependStep.Output))
			case model.StepFailed, model.StepCanceled, model.StepSkipped:
				err = store.Chatbot.UpdateStepState(int64(step.ID), dependStep.State)
				if err != nil {
					flog.Error(err)
				}
				allFinished = false
			}
		}
		if allFinished {
			err = store.Chatbot.UpdateStepState(int64(step.ID), model.StepReady)
			if err != nil {
				flog.Error(err)
			}
			// update input
			err = store.Chatbot.UpdateStepInput(int64(step.ID), mergeOutput)
			if err != nil {
				flog.Error(err)
			}
			// update started at
			err = store.Chatbot.UpdateStepStartedAt(int64(step.ID), time.Now())
			if err != nil {
				flog.Error(err)
			}
		}
	}
}

func KeyFunc(obj interface{}) (string, error) {
	if j, ok := obj.(*types.StepInfo); ok {
		return stepKey(j.Step), nil
	}
	return "", errors.New("unknown object")
}

func stepKey(step *model.Step) string {
	return fmt.Sprintf("step-%d", step.ID)
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

				err := store.Chatbot.UpdateStepState(int64(step.ID), model.StepRunning)
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

				err := store.Chatbot.UpdateStepState(int64(step.ID), model.StepFinished)
				if err != nil {
					e.Cancel(err)
					return
				}
				// update finished at
				err = store.Chatbot.UpdateStepFinishedAt(int64(step.ID), time.Now())
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

				err := store.Chatbot.UpdateStepState(int64(step.ID), model.StepFailed)
				if err != nil {
					e.Cancel(err)
					return
				}
			},
		},
	)

	return f
}
