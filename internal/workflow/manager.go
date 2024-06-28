package workflow

import (
	"context"
	"time"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/utils/parallelizer"
	"github.com/hibiken/asynq"
	"github.com/pkg/errors"
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
	// sync job
	go parallelizer.JitterUntil(m.syncJob, time.Minute, 0.0, true, m.stop)
	// ready job
	go parallelizer.JitterUntil(m.pushReadyJob, time.Second, 0.0, true, m.stop)
}

func (m *Manager) Shutdown() {
	m.stop <- struct{}{}
}

func (m *Manager) syncJob() {
	list, err := store.Database.GetJobsByState(model.JobReady)
	if err != nil {
		flog.Error(err)
	}
	for _, job := range list {
		err = SyncJob(context.Background(), job)
		if err != nil {
			flog.Error(errors.Wrapf(err, "failed to sync job %d", job.ID))
		}
	}
}

func (m *Manager) pushReadyJob() {
	ctx := context.Background()
	list, err := GetJobsByState(ctx, model.JobReady)
	if err != nil {
		flog.Error(err)
		return
	}
	for _, job := range list {
		job.State = model.JobStart
		t, err := NewJobTask(job)
		if err != nil {
			flog.Error(errors.Wrapf(err, "failed to create task for job %d", job.ID))
			continue
		}
		err = PushTask(t)
		if err != nil {
			// duplicate task
			if errors.Is(err, asynq.ErrDuplicateTask) {
				flog.Warn("duplicate task: %s, skip", t.ID)
				continue
			}

			// task id conflict
			if errors.Is(err, asynq.ErrTaskIDConflict) {
				flog.Warn("task id conflict: %s, skip", t.ID)

				err = DeleteJob(context.Background(), job)
				if err != nil {
					flog.Error(errors.Wrapf(err, "failed to delete job %d", job.ID))
				}

				err = store.Database.UpdateJobState(job.ID, model.JobFailed)
				if err != nil {
					flog.Error(errors.Wrapf(err, "failed to update job %d", job.ID))
				}

				continue
			}

			flog.Error(errors.Wrapf(err, "failed to push task %s", t.ID))
			continue
		}
		err = store.Database.UpdateJobState(job.ID, model.JobStart)
		if err != nil {
			flog.Error(errors.Wrapf(err, "failed to update job %d", job.ID))
			continue
		}
		err = SyncJob(ctx, job)
		if err != nil {
			flog.Error(errors.Wrapf(err, "failed to sync job %d", job.ID))
			continue
		}
	}
}
