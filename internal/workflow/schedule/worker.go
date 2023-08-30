package schedule

import (
	"context"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/utils/queue"
)

type Worker struct {
	Queue *queue.DeltaFIFO

	stop chan struct{}
}

func NewWorker(queue *queue.DeltaFIFO) *Worker {
	return &Worker{
		Queue: queue,
		stop:  make(chan struct{}),
	}
}

func (m *Worker) Run() {
	for {
		select {
		case <-m.stop:
			flog.Info("worker stopped")
			return
		default:
			m.popStep()
		}
	}
}

func (m *Worker) Shutdown() {
	m.stop <- struct{}{}
}

func (m *Worker) popStep() {
	_, err := m.Queue.Pop(func(i interface{}) error {
		if d, ok := i.(queue.Deltas); ok {
			for _, delta := range d {
				if delta.Type != queue.Added {
					return nil
				}
				if j, ok := delta.Object.(*types.StepInfo); ok {
					err := j.FSM.Event(context.Background(), "run", j.Step)
					if err != nil {
						flog.Error(err)
						_ = j.FSM.Event(context.Background(), "error", j.Step)
					} else {
						_ = j.FSM.Event(context.Background(), "success", j.Step)
					}
				}
			}
		}
		return nil
	})
	if err != nil {
		flog.Error(err)
	}
}
