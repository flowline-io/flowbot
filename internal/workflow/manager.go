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
	go parallelizer.JitterUntil(m.pushReadyJob, time.Second, 0.0, true, m.stop)

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
	list, err := store.Database.GetJobsByState(model.JobReady)
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
		err = store.Database.UpdateJobState(job.ID, model.JobStart)
		if err != nil {
			flog.Error(err)
			continue
		}
	}
}
