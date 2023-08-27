package manager

import (
	"context"
	"errors"
	"fmt"
	"github.com/sysatom/flowbot/internal/store"
	"github.com/sysatom/flowbot/internal/store/model"
	"github.com/sysatom/flowbot/pkg/dag"
	"github.com/sysatom/flowbot/pkg/logs"
	"github.com/sysatom/flowbot/pkg/utils/parallelizer"
	"github.com/sysatom/flowbot/pkg/utils/queue"
	"time"
)

type Manager struct {
	Queue *queue.DeltaFIFO

	stop chan struct{}
}

func NewManager() *Manager {
	return &Manager{
		Queue: queue.NewDeltaFIFOWithOptions(queue.DeltaFIFOOptions{
			KeyFunction: JobKeyFunc,
		}),
		stop: make(chan struct{}),
	}
}

func (m *Manager) Run(ctx context.Context) {

	go parallelizer.JitterUntil(m.pushJob, time.Second, 0.0, true, m.stop)

	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stop:
			logs.Info.Println("manager stopped")
			return
		default:
			m.popJob()
		}
	}

}

func (m *Manager) Shutdown() {
	m.stop <- struct{}{}
}

func (m *Manager) manager() {
	fmt.Println("manager", time.Now().UnixMicro())
}

func (m *Manager) pushJob() {
	fmt.Println("manager", time.Now().UnixMicro())

	list, err := store.Chatbot.GetJobsByState(model.JobReady)
	if err != nil {
		logs.Err.Println(err)
		return
	}
	for _, job := range list {
		_, exists, err := m.Queue.Get(job)
		if err != nil {
			logs.Err.Println(err)
			continue
		}
		if exists {
			continue
		}

		err = m.Queue.Add(job)
		if err != nil {
			logs.Err.Println(err)
		}
	}
}

func (m *Manager) popJob() {
	_, err := m.Queue.Pop(func(i interface{}) error {
		if d, ok := i.(queue.Deltas); ok {
			for _, delta := range d {
				if delta.Type != queue.Added {
					return nil
				}
				if j, ok := delta.Object.(*model.Job); ok {
					return m.splitDag(j)
				}
			}
		}
		return nil
	})
	if err != nil {
		logs.Err.Println(err)
	}
}

func (m *Manager) splitDag(job *model.Job) error {
	d, err := store.Chatbot.GetDag(int64(job.DagID))
	if err != nil {
		return err
	}
	list, err := dag.TopologySort(d)
	if err != nil {
		return err
	}

	// create steps
	steps := make([]*model.Step, len(list))
	for _, step := range list {
		steps = append(steps, &model.Step{
			UID:    job.UID,
			Topic:  job.Topic,
			JobID:  job.ID,
			Action: "", // todo
			Name:   step.Name,
			State:  step.State,
		})
	}
	err = store.Chatbot.CreateSteps(steps)
	if err != nil {
		return err
	}

	// update job state
	return store.Chatbot.UpdateJobState(int64(job.ID), model.JobStart)
}

func JobKeyFunc(obj interface{}) (string, error) {
	if j, ok := obj.(*model.Job); ok {
		return fmt.Sprintf("job-%d", j.ID), nil
	}
	return "", errors.New("unknown object")
}
