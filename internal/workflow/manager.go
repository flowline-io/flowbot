package workflow

import (
	"context"
	"errors"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/pkg/dag"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/utils/parallelizer"
	"github.com/looplab/fsm"
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
	go parallelizer.JitterUntil(m.pushReadyJob, time.Second, 0.0, true, m.stop)
	// check job
	go parallelizer.JitterUntil(m.checkJob, 10*time.Second, 0.0, true, m.stop)

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
		t, err := NewJobTask(job)
		if err != nil {
			flog.Error(err)
			continue
		}
		err = PushTask(t)
		if err != nil {
			flog.Error(err)
		}
	}
}

func (m *Manager) checkJob() {
	list, err := store.Chatbot.GetJobsByState(model.JobStart)
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
		lastFinishedAt := time.Now().AddDate(-1000, 0, 0)
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
