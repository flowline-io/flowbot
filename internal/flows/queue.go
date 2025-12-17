package flows

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/lithammer/shortuuid/v4"
	"gorm.io/gorm"
)

const (
	flowQueueStatusPending   int8 = 0
	flowQueueStatusRunning   int8 = 1
	flowQueueStatusCompleted int8 = 2
	flowQueueStatusFailed    int8 = 3
)

// FlowQueueJob represents a MySQL-backed flow execution queue job.
// Table is created by migration: 000050_create_flow_queue_jobs_table.
type FlowQueueJob struct {
	ID          uint64   `gorm:"primaryKey;autoIncrement"`
	FlowID      int64    `gorm:"not null;index:idx_flow_queue_status_runat"`
	ExecutionID string   `gorm:"type:varchar(64);not null;default:'';index:idx_flow_queue_execution_id"`
	TriggerType string   `gorm:"type:varchar(32);not null"`
	TriggerID   string   `gorm:"type:varchar(128);not null"`
	Payload     types.KV `gorm:"type:json"`
	Status      int8     `gorm:"not null;index:idx_flow_queue_status_runat"`
	Attempts    int      `gorm:"not null"`
	MaxAttempts int      `gorm:"not null;default:3"`
	RunAt       time.Time
	LockedAt    *time.Time
	LastError   string `gorm:"type:text"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (FlowQueueJob) TableName() string { return "flow_queue_jobs" }

// QueueManager manages a simple MySQL-backed queue for flow executions.
type QueueManager struct {
	db      *gorm.DB
	store   store.Adapter
	engine  *Engine
	workers int

	mu     sync.Mutex
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewQueueManager creates a new queue manager
func NewQueueManager(storeAdapter store.Adapter, engine *Engine) (*QueueManager, error) {
	db := storeAdapter.GetDB()
	if db == nil {
		return nil, errors.New("flow queue manager: store DB is nil")
	}

	return &QueueManager{
		db:      db,
		store:   storeAdapter,
		engine:  engine,
		workers: 2,
	}, nil
}

// Start starts the queue manager
func (q *QueueManager) Start(ctx context.Context) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.cancel != nil {
		return nil
	}

	// IMPORTANT: the Fx OnStart context is usually canceled as soon as startup
	// completes. Use a long-lived context for workers and rely on Stop() to cancel.
	ctx, cancel := context.WithCancel(context.Background())
	q.cancel = cancel

	workers := q.workers
	if workers <= 0 {
		workers = 1
	}
	for i := 0; i < workers; i++ {
		q.wg.Add(1)
		go func(workerID int) {
			defer q.wg.Done()
			q.workerLoop(ctx, workerID)
		}(i + 1)
	}

	flog.Info("flow queue manager: started (mysql)")
	return nil
}

// Stop stops the queue manager
func (q *QueueManager) Stop(ctx context.Context) error {
	q.mu.Lock()
	cancel := q.cancel
	q.cancel = nil
	q.mu.Unlock()

	if cancel != nil {
		cancel()
	}

	done := make(chan struct{})
	go func() {
		q.wg.Wait()
		close(done)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
		return nil
	}
}

// EnqueueFlowExecution enqueues a flow execution job
// Currently returns error as queue is disabled - use synchronous execution instead
func (q *QueueManager) EnqueueFlowExecution(ctx context.Context, flowID int64, triggerType string, triggerID string, payload types.KV) (string, error) {
	flow, err := q.store.GetFlow(flowID)
	if err != nil {
		return "", fmt.Errorf("failed to get flow: %w", err)
	}
	if !flow.Enabled {
		return "", fmt.Errorf("flow is disabled")
	}

	executionID := shortuuid.New()
	job := &FlowQueueJob{
		FlowID:      flowID,
		ExecutionID: executionID,
		TriggerType: triggerType,
		TriggerID:   triggerID,
		Payload:     payload,
		Status:      flowQueueStatusPending,
		Attempts:    0,
		MaxAttempts: 3,
		RunAt:       time.Now(),
	}
	if err := q.db.WithContext(ctx).Create(job).Error; err != nil {
		return "", fmt.Errorf("failed to enqueue flow job: %w", err)
	}
	flog.Info("enqueued flow execution job %d for flow %d (execution %s)", job.ID, flowID, executionID)
	return executionID, nil
}

func (q *QueueManager) workerLoop(ctx context.Context, workerID int) {
	baseIdleDelay := 500 * time.Millisecond
	maxIdleDelay := 5 * time.Second
	idleDelay := baseIdleDelay

	rng := rand.New(rand.NewSource(time.Now().UnixNano() + int64(workerID)))
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		job, ok, err := q.claimNextJob(ctx)
		if err != nil {
			flog.Error(fmt.Errorf("flow queue worker %d: claim failed: %w", workerID, err))
			// Avoid hot-looping on DB errors.
			time.Sleep(time.Second + time.Duration(rng.Intn(250))*time.Millisecond)
			continue
		}
		if !ok {
			// Adaptive backoff to reduce DB polling when the queue is empty.
			jitter := time.Duration(rng.Intn(250)) * time.Millisecond
			time.Sleep(idleDelay + jitter)
			if idleDelay < maxIdleDelay {
				idleDelay *= 2
				if idleDelay > maxIdleDelay {
					idleDelay = maxIdleDelay
				}
			}
			continue
		}
		idleDelay = baseIdleDelay

		_, err = q.engine.ExecuteFlowWithExecutionID(ctx, job.FlowID, job.ExecutionID, job.TriggerType, job.TriggerID, job.Payload)
		if err == nil {
			_ = q.db.WithContext(ctx).
				Model(&FlowQueueJob{}).
				Where("id = ?", job.ID).
				Updates(map[string]any{
					"status":     flowQueueStatusCompleted,
					"updated_at": time.Now(),
				}).Error
			continue
		}

		attempts := job.Attempts + 1
		updates := map[string]any{
			"attempts":   attempts,
			"last_error": err.Error(),
			"updated_at": time.Now(),
		}

		if attempts >= job.MaxAttempts {
			updates["status"] = flowQueueStatusFailed
			_ = q.db.WithContext(ctx).
				Model(&FlowQueueJob{}).
				Where("id = ?", job.ID).
				Updates(updates).Error
			continue
		}

		// Retry later with a simple exponential backoff.
		delay := time.Duration(1<<min(attempts, 6)) * time.Second
		updates["status"] = flowQueueStatusPending
		updates["run_at"] = time.Now().Add(delay)
		updates["locked_at"] = nil
		_ = q.db.WithContext(ctx).
			Model(&FlowQueueJob{}).
			Where("id = ?", job.ID).
			Updates(updates).Error
	}
}

func (q *QueueManager) claimNextJob(ctx context.Context) (*FlowQueueJob, bool, error) {
	if q.db == nil {
		return nil, false, errors.New("queue db is nil")
	}

	var claimed *FlowQueueJob
	err := q.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var job FlowQueueJob
		now := time.Now()
		// MySQL 8+: SKIP LOCKED prevents thundering herds.
		// If not available, this will error and we'll fall back to synchronous in API.
		if err := tx.Raw(
			"SELECT * FROM flow_queue_jobs WHERE status = ? AND run_at <= ? ORDER BY id ASC LIMIT 1 FOR UPDATE SKIP LOCKED",
			flowQueueStatusPending,
			now,
		).Scan(&job).Error; err != nil {
			return err
		}
		if job.ID == 0 {
			return nil
		}
		res := tx.Exec(
			"UPDATE flow_queue_jobs SET status = ?, locked_at = ?, updated_at = ? WHERE id = ? AND status = ?",
			flowQueueStatusRunning,
			now,
			now,
			job.ID,
			flowQueueStatusPending,
		)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return nil
		}

		job.Status = flowQueueStatusRunning
		job.LockedAt = &now
		claimed = &job
		return nil
	})
	if err != nil {
		return nil, false, err
	}
	if claimed == nil {
		return nil, false, nil
	}
	return claimed, true, nil
}
