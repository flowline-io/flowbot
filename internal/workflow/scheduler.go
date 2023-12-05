package workflow

import (
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/utils/parallelizer"
	"time"
)

type Scheduler struct {
	stop chan struct{}
}

func NewScheduler() *Scheduler {
	s := &Scheduler{
		stop: make(chan struct{}),
	}
	return s
}

func (sched *Scheduler) Run() {
	// ready step
	go parallelizer.JitterUntil(sched.pushReadyStep, time.Second, 0.0, true, sched.stop)
	// depend step
	go parallelizer.JitterUntil(sched.dependStep, time.Second, 0.0, true, sched.stop)

	<-sched.stop
	flog.Info("scheduler stopped")
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
		step.State = model.StepStart
		t, err := NewWorkerTask(step)
		if err != nil {
			flog.Error(err)
			continue
		}
		err = PushTask(t)
		if err != nil {
			flog.Error(err)
			continue
		}
		err = store.Chatbot.UpdateStepState(step.ID, model.StepStart)
		if err != nil {
			flog.Error(err)
			continue
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
		dependSteps, err := store.Chatbot.GetStepsByDepend(step.JobID, step.Depend)
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
				err = store.Chatbot.UpdateStepState(step.ID, dependStep.State)
				if err != nil {
					flog.Error(err)
				}
				allFinished = false
			}
		}
		if allFinished {
			err = store.Chatbot.UpdateStepState(step.ID, model.StepReady)
			if err != nil {
				flog.Error(err)
			}
			// update input
			err = store.Chatbot.UpdateStepInput(step.ID, mergeOutput)
			if err != nil {
				flog.Error(err)
			}
			// update started at
			err = store.Chatbot.UpdateStepStartedAt(step.ID, time.Now())
			if err != nil {
				flog.Error(err)
			}
		}
	}
}
