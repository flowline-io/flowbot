package flows

import (
	"context"
	"errors"
	"fmt"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
)

// FlowExecutionJob represents a flow execution job
type FlowExecutionJob struct {
	FlowID      int64    `json:"flow_id"`
	TriggerType string   `json:"trigger_type"`
	TriggerID   string   `json:"trigger_id"`
	Payload     types.KV `json:"payload"`
}

func (FlowExecutionJob) Kind() string {
	return "flow_execution"
}

func (j FlowExecutionJob) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		MaxAttempts: 3,
	}
}

// FlowExecutionWorker executes flow execution jobs
type FlowExecutionWorker struct {
	engine *Engine
	river.WorkerDefaults[FlowExecutionJob]
}

func NewFlowExecutionWorker(engine *Engine) *FlowExecutionWorker {
	return &FlowExecutionWorker{
		engine: engine,
	}
}

func (w *FlowExecutionWorker) Work(ctx context.Context, job *river.Job[FlowExecutionJob]) error {
	flog.Info("executing flow %d", job.Args.FlowID)

	err := w.engine.ExecuteFlow(ctx, job.Args.FlowID, job.Args.TriggerType, job.Args.TriggerID, job.Args.Payload)
	if err != nil {
		return fmt.Errorf("failed to execute flow: %w", err)
	}

	return nil
}

// QueueManager manages the river queue for flow executions
// Note: riverqueue doesn't have direct MySQL support, so we use synchronous execution
// In production, you might want to use a different queue system or add MySQL support
type QueueManager struct {
	client  *river.Client[*rivertype.JobRow]
	store   store.Adapter
	engine  *Engine
	workers *river.Workers
}

// NewQueueManager creates a new queue manager
func NewQueueManager(storeAdapter store.Adapter, engine *Engine) (*QueueManager, error) {
	workers := river.NewWorkers()
	worker := NewFlowExecutionWorker(engine)
	river.AddWorker(workers, worker)

	// Queue is disabled for now - riverqueue doesn't have direct MySQL support
	// We'll use synchronous execution instead
	return &QueueManager{
		client:  nil, // Queue disabled - using synchronous execution
		store:   storeAdapter,
		engine:  engine,
		workers: workers,
	}, nil
}

// Start starts the queue manager
func (q *QueueManager) Start(ctx context.Context) error {
	// Queue is disabled for now - riverqueue doesn't have direct MySQL support
	// We'll use synchronous execution instead
	flog.Info("flow queue manager: using synchronous execution (queue disabled)")
	return nil
}

// Stop stops the queue manager
func (q *QueueManager) Stop(ctx context.Context) error {
	if q.client == nil {
		return nil
	}
	return q.client.Stop(ctx)
}

// EnqueueFlowExecution enqueues a flow execution job
// Currently returns error as queue is disabled - use synchronous execution instead
func (q *QueueManager) EnqueueFlowExecution(ctx context.Context, flowID int64, triggerType string, triggerID string, payload types.KV) error {
	if q.client == nil {
		return errors.New("queue is disabled - use synchronous execution")
	}

	job := FlowExecutionJob{
		FlowID:      flowID,
		TriggerType: triggerType,
		TriggerID:   triggerID,
		Payload:     payload,
	}

	result, err := q.client.Insert(ctx, job, nil)
	if err != nil {
		return fmt.Errorf("failed to insert job: %w", err)
	}

	flog.Info("enqueued flow execution job %d for flow %d", result.Job.ID, flowID)
	return nil
}

// ErrorHandler handles job errors
type ErrorHandler struct{}

func (h *ErrorHandler) HandleError(ctx context.Context, job *rivertype.JobRow, err error) *river.ErrorHandlerResult {
	flog.Error(fmt.Errorf("job %d failed: %w", job.ID, err))
	return nil
}

func (h *ErrorHandler) HandlePanic(ctx context.Context, job *rivertype.JobRow, panicVal any, trace string) *river.ErrorHandlerResult {
	flog.Error(fmt.Errorf("job %d panicked: %v", job.ID, panicVal))
	flog.Warn("Stack trace: %s", trace)
	return nil
}
