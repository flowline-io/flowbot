package workflow

import (
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/utils/parallelizer"
	"time"
)

type Manager struct {
	stop chan struct{}
}

func NewManager() *Manager {
	return &Manager{
		stop: make(chan struct{}),
	}
}

func (m *Manager) Run() {
	// ready job
	go parallelizer.JitterUntil(m.pushReadyJob, 2*time.Second, 0.0, true, m.stop)
	// check job
	go parallelizer.JitterUntil(m.checkJob, 2*time.Second, 0.0, true, m.stop)

	for {
		select {
		case <-m.stop:
			flog.Info("manager stopped")
			return
		}
	}
}

func (m *Manager) Shutdown() {
	m.stop <- struct{}{}
}

func (m *Manager) pushReadyJob() {
	list, err := store.Chatbot.GetJobsByState(model.JobReady)
	if err != nil {
		flog.Error(err)
		return
	}
	for _, job := range list {
		job.State = model.JobStart
		t, err := NewJobTask(job)
		if err != nil {
			flog.Error(err)
			continue
		}
		err = PushTask(t)
		if err != nil {
			flog.Error(err)
			continue
		}
		err = store.Chatbot.UpdateJobState(job.ID, model.JobStart)
		if err != nil {
			flog.Error(err)
			continue
		}
	}
}

func (m *Manager) checkJob() {
	list, err := store.Chatbot.GetJobsByState(model.JobRunning)
	if err != nil {
		flog.Error(err)
		return
	}
	for _, job := range list {
		steps, err := store.Chatbot.GetStepsByJobId(job.ID)
		if err != nil {
			flog.Error(err)
			continue
		}
		allFinished := true
		keeping := false
		canceled := false
		failed := false
		lastFinishedAt := time.Time{}
		for _, step := range steps {
			switch step.State {
			case model.StepCreated, model.StepReady, model.StepRunning:
				keeping = true
				allFinished = false
			case model.StepFinished, model.StepSkipped:
				if step.FinishedAt != nil && step.FinishedAt.After(lastFinishedAt) {
					lastFinishedAt = *step.FinishedAt
				}
			case model.StepFailed:
				failed = true
				allFinished = false
			case model.StepCanceled:
				canceled = true
				allFinished = false
			}
		}
		if keeping {
			continue
		}
		if allFinished {
			err = store.Chatbot.UpdateJobState(job.ID, model.JobFinished)
			if err != nil {
				flog.Error(err)
			}
			// update finished at
			err = store.Chatbot.UpdateJobFinishedAt(job.ID, lastFinishedAt)
			if err != nil {
				flog.Error(err)
			}
			// successful count
			err = store.Chatbot.IncreaseWorkflowCount(job.WorkflowID, 1, 0, -1, 0)
			if err != nil {
				flog.Error(err)
			}
			continue
		}
		if failed {
			err = store.Chatbot.UpdateJobState(job.ID, model.JobFailed)
			if err != nil {
				flog.Error(err)
			}
			// failed count
			err = store.Chatbot.IncreaseWorkflowCount(job.WorkflowID, 0, 1, -1, 0)
			if err != nil {
				flog.Error(err)
			}
			continue
		}
		if canceled {
			err = store.Chatbot.UpdateJobState(job.ID, model.JobCanceled)
			if err != nil {
				flog.Error(err)
			}
			// canceled count
			err = store.Chatbot.IncreaseWorkflowCount(job.WorkflowID, 0, 0, -1, 1)
			if err != nil {
				flog.Error(err)
			}
			continue
		}
	}
}
